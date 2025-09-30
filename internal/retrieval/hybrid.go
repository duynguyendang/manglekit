package retrieval

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"

	"ndduy.dev/manglekit/internal/types"
)

type hybridRetriever struct {
	cfg          Config
	chunks       []*chunkIndex
	idf          map[string]float64
	avgDocLength float64
	denseStore   *localvec.DocStore
	docIndex     map[string]*chunkIndex
}

type chunkIndex struct {
	chunk    *types.Chunk
	tokens   []string
	termFreq map[string]float64
	length   float64
}

type candidateScore struct {
	chunk    *types.Chunk
	lexical  float64
	dense    float64
	combined float64
}

// NewHybrid constructs a hybrid retriever that performs lexical and dense search.
func NewHybrid(ctx context.Context, g *genkit.Genkit, embedder ai.Embedder, cfg Config) (types.Retriever, error) {
	if cfg.Corpus.Path == "" {
		return nil, errors.New("retrieval corpus path is required")
	}
	if cfg.Hybrid.Dense.TopK <= 0 {
		cfg.Hybrid.Dense.TopK = 20
	}
	if cfg.Hybrid.BM25.Must < 0 {
		cfg.Hybrid.BM25.Must = 0
	}
	if cfg.Hybrid.BM25.Should < 0 {
		cfg.Hybrid.BM25.Should = 0
	}
	if cfg.Corpus.ChunkSize <= 0 {
		cfg.Corpus.ChunkSize = 180
	}
	if cfg.Corpus.ChunkOverlap < 0 {
		cfg.Corpus.ChunkOverlap = 0
	}

	if g == nil {
		return nil, errors.New("genkit instance is required for dense retrieval")
	}

	if embedder == nil {
		return nil, errors.New("embedder is required for dense retrieval")
	}

	if err := localvec.Init(); err != nil {
		return nil, fmt.Errorf("init localvec: %w", err)
	}

	collectionName := "hybrid-dense"
	docStore, _, err := localvec.DefineRetriever(g, collectionName, localvec.Config{
		Dir:      cfg.Hybrid.Dense.StoreDir,
		Embedder: embedder,
	}, &ai.RetrieverOptions{Label: collectionName})
	if err != nil {
		return nil, fmt.Errorf("define localvec retriever: %w", err)
	}

	chunks, err := loadCorpus(cfg)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks were indexed from %s", cfg.Corpus.Path)
	}

	docs := make([]*ai.Document, 0, len(chunks))
	docIndex := make(map[string]*chunkIndex, len(chunks))
	for _, idx := range chunks {
		meta := map[string]any{
			"chunkID": idx.chunk.ID,
		}
		doc := ai.DocumentFromText(idx.chunk.Text, meta)
		docs = append(docs, doc)
		docID, err := localVecDocID(doc)
		if err != nil {
			return nil, fmt.Errorf("compute document id for chunk %s: %w", idx.chunk.ID, err)
		}
		docIndex[docID] = idx
	}
	if err := localvec.Index(ctx, docs, docStore); err != nil {
		return nil, fmt.Errorf("index corpus with localvec: %w", err)
	}

	idf, avgLen := computeIDF(chunks)

	return &hybridRetriever{
		cfg:          cfg,
		chunks:       chunks,
		idf:          idf,
		avgDocLength: avgLen,
		denseStore:   docStore,
		docIndex:     docIndex,
	}, nil
}

func (r *hybridRetriever) Search(ctx context.Context, query *types.ExpandedQuery, filters map[string]string) ([]*types.Chunk, error) {
	if query == nil {
		return nil, errors.New("expanded query is required")
	}

	mustTerms := projectTerms(query.Constraints.Terms.Must, r.cfg.Hybrid.BM25.Must)
	if len(mustTerms) == 0 {
		mustTerms = projectTerms(query.NormalizedTerms, r.cfg.Hybrid.BM25.Must)
	}
	shouldTerms := projectTerms(query.Constraints.Terms.Should, r.cfg.Hybrid.BM25.Should)
	if len(shouldTerms) == 0 {
		shouldTerms = projectTerms(query.ExpansionTerms, r.cfg.Hybrid.BM25.Should)
	}
	shouldTerms = append(shouldTerms, flattenEntityValues(query.Entities)...)

	metadataFilters := mergeFilters(filters, query)

	queryText := strings.Join(append([]string{query.NormalizedQuery}, append(mustTerms, shouldTerms...)...), " ")
	denseScores := map[string]float64{}
	if r.denseStore != nil {
		scores, err := r.computeDenseScores(ctx, queryText)
		if err != nil {
			return nil, fmt.Errorf("compute dense scores: %w", err)
		}
		denseScores = scores
	}

	var candidates []candidateScore
	for _, idx := range r.chunks {
		if !matchesMetadata(idx.chunk.Metadata, metadataFilters) {
			continue
		}
		if len(mustTerms) > 0 && !containsAll(idx.tokens, mustTerms) {
			continue
		}
		lexicalScore := r.scoreBM25(idx, append(mustTerms, shouldTerms...))
		denseScore := denseScores[idx.chunk.ID]
		if lexicalScore == 0 && denseScore == 0 {
			continue
		}
		candidates = append(candidates, candidateScore{
			chunk:   idx.cloneChunk(),
			lexical: lexicalScore,
			dense:   denseScore,
		})
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	limit := r.cfg.Hybrid.Dense.TopK
	lexicalOrder := rankCandidates(candidates, func(a, b candidateScore) bool { return a.lexical > b.lexical })
	denseOrder := rankCandidates(candidates, func(a, b candidateScore) bool { return a.dense > b.dense })

	selected := make(map[string]*candidateScore)
	for i := 0; i < len(lexicalOrder) && i < limit; i++ {
		selected[lexicalOrder[i].chunk.ID] = lexicalOrder[i]
	}
	for i := 0; i < len(denseOrder) && i < limit; i++ {
		selected[denseOrder[i].chunk.ID] = denseOrder[i]
	}

	if len(selected) == 0 {
		return nil, nil
	}

	var maxLex, maxDense float64
	for _, sc := range selected {
		if sc.lexical > maxLex {
			maxLex = sc.lexical
		}
		if sc.dense > maxDense {
			maxDense = sc.dense
		}
	}
	for _, sc := range selected {
		var lexNorm, denseNorm float64
		if maxLex > 0 {
			lexNorm = sc.lexical / maxLex
		}
		if maxDense > 0 {
			denseNorm = sc.dense / maxDense
		}
		sc.combined = 0.6*lexNorm + 0.4*denseNorm
		if sc.chunk.Metadata == nil {
			sc.chunk.Metadata = map[string]any{}
		}
		sc.chunk.Metadata["lexicalScore"] = sc.lexical
		sc.chunk.Metadata["denseScore"] = sc.dense
		sc.chunk.Metadata["hybridScore"] = sc.combined
		sc.chunk.Score = sc.combined
	}

	ranked := make([]*candidateScore, 0, len(selected))
	for _, sc := range selected {
		ranked = append(ranked, sc)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].combined == ranked[j].combined {
			return ranked[i].chunk.ID < ranked[j].chunk.ID
		}
		return ranked[i].combined > ranked[j].combined
	})

	results := make([]*types.Chunk, 0, len(ranked))
	for _, sc := range ranked {
		results = append(results, sc.chunk)
	}
	return results, nil
}

func (r *hybridRetriever) scoreBM25(idx *chunkIndex, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	const (
		k1 = 1.6
		b  = 0.75
	)
	var score float64
	seen := make(map[string]struct{})
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		tf := idx.termFreq[term]
		if tf == 0 {
			continue
		}
		idf := r.idf[term]
		denom := tf + k1*(1-b+b*(idx.length/r.avgDocLength))
		score += idf * (tf * (k1 + 1) / denom)
	}
	return score
}

func (r *hybridRetriever) computeDenseScores(ctx context.Context, queryText string) (map[string]float64, error) {
	if r.denseStore == nil || r.denseStore.Embedder == nil {
		return map[string]float64{}, nil
	}
	if len(r.denseStore.Data) == 0 {
		return map[string]float64{}, nil
	}

	queryDoc := ai.DocumentFromText(queryText, nil)
	embedReq := &ai.EmbedRequest{
		Input:   []*ai.Document{queryDoc},
		Options: r.denseStore.EmbedderOptions,
	}
	embedResp, err := r.denseStore.Embedder.Embed(ctx, embedReq)
	if err != nil {
		return nil, fmt.Errorf("embed query with google ai: %w", err)
	}
	if len(embedResp.Embeddings) == 0 {
		return map[string]float64{}, nil
	}
	queryVec := embedResp.Embeddings[0].Embedding

	scores := make(map[string]float64, len(r.denseStore.Data))
	for docID, stored := range r.denseStore.Data {
		idx := r.docIndex[docID]
		if idx == nil {
			continue
		}
		scores[idx.chunk.ID] = cosineSimilarity32(queryVec, stored.Embedding)
	}
	return scores, nil
}

func rankCandidates(candidates []candidateScore, less func(a, b candidateScore) bool) []*candidateScore {
	order := make([]*candidateScore, 0, len(candidates))
	for i := range candidates {
		order = append(order, &candidates[i])
	}
	sort.Slice(order, func(i, j int) bool {
		return less(*order[i], *order[j])
	})
	return order
}

func mergeFilters(base map[string]string, query *types.ExpandedQuery) map[string][]string {
	merged := make(map[string][]string)
	for k, v := range base {
		if strings.TrimSpace(k) == "" {
			continue
		}
		merged[strings.ToLower(k)] = append(merged[strings.ToLower(k)], strings.ToLower(v))
	}
	for _, c := range query.Constraints.Metadata {
		if c.Field == "" {
			continue
		}
		merged[strings.ToLower(c.Field)] = append(merged[strings.ToLower(c.Field)], c.Values...)
	}
	if query.Constraints.Visibility != "" {
		merged["visibility"] = append(merged["visibility"], strings.ToLower(query.Constraints.Visibility))
	}
	return merged
}

func matchesMetadata(meta map[string]any, filters map[string][]string) bool {
	if len(filters) == 0 {
		return true
	}
	for key, values := range filters {
		if len(values) == 0 {
			continue
		}
		val := strings.ToLower(asString(meta[key]))
		matched := false
		for _, want := range values {
			want = strings.ToLower(want)
			if want == "*" || val == want {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func containsAll(tokens []string, must []string) bool {
	if len(must) == 0 {
		return true
	}
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = struct{}{}
	}
	for _, term := range must {
		term = strings.ToLower(term)
		if _, ok := tokenSet[term]; !ok {
			return false
		}
	}
	return true
}

func projectTerms(source []string, limit int) []string {
	if len(source) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(source))
	var result []string
	for _, term := range source {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		result = append(result, term)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func flattenEntityValues(entities map[string][]string) []string {
	if len(entities) == 0 {
		return nil
	}
	var values []string
	for _, v := range entities {
		values = append(values, v...)
	}
	return values
}

func loadCorpus(cfg Config) ([]*chunkIndex, error) {
	var files []string
	if err := filepath.WalkDir(cfg.Corpus.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)

	var chunks []*chunkIndex
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		docID := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		title, body := splitTitle(string(data))
		ch := chunkDocument(docID, title, body, cfg.Corpus)
		for _, chunk := range ch {
			idx := &chunkIndex{
				chunk:    chunk,
				tokens:   tokenize(strings.ToLower(chunk.Text)),
				termFreq: make(map[string]float64),
			}
			for _, token := range idx.tokens {
				idx.termFreq[token]++
			}
			idx.length = float64(len(idx.tokens))
			chunks = append(chunks, idx)
		}
	}
	return chunks, nil
}

func chunkDocument(docID, title, body string, cfg CorpusConfig) []*types.Chunk {
	tokens := tokenize(body)
	if len(tokens) == 0 {
		return []*types.Chunk{
			{
				ID:    docID + "#0",
				DocID: docID,
				Title: title,
				Text:  body,
				Metadata: map[string]any{
					"visibility": "public",
					"tenant":     "*",
					"createdAt":  time.Now().Format(time.RFC3339),
				},
			},
		}
	}
	size := cfg.ChunkSize
	overlap := cfg.ChunkOverlap
	if size <= 0 {
		size = 180
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 2
	}
	step := size - overlap
	if step <= 0 {
		step = size
	}

	var chunks []*types.Chunk
	start := 0
	index := 0
	for start < len(tokens) {
		end := start + size
		if end > len(tokens) {
			end = len(tokens)
		}
		segment := tokens[start:end]
		text := strings.Join(segment, " ")
		chunk := &types.Chunk{
			ID:    fmt.Sprintf("%s#%d", docID, index+1),
			DocID: docID,
			Title: title,
			Text:  text,
			Snippet: func() string {
				limit := 40
				if len(segment) < limit {
					limit = len(segment)
				}
				return strings.Join(segment[:limit], " ")
			}(),
			Metadata: map[string]any{
				"visibility": "public",
				"tenant":     "*",
				"docPath":    docID,
				"position":   index,
			},
		}
		chunks = append(chunks, chunk)
		if end == len(tokens) {
			break
		}
		start += step
		index++
	}
	return chunks
}

func splitTitle(body string) (string, string) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimLeft(trimmed, "# "), strings.Join(lines, "\n")
		}
		break
	}
	return "Untitled", body
}

func tokenize(text string) []string {
	var tokens []string
	var builder strings.Builder
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(builder.String()))
		builder.Reset()
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func computeIDF(chunks []*chunkIndex) (map[string]float64, float64) {
	docFreq := make(map[string]float64)
	var totalLen float64
	for _, idx := range chunks {
		seen := make(map[string]struct{})
		for _, token := range idx.tokens {
			seen[token] = struct{}{}
		}
		for token := range seen {
			docFreq[token]++
		}
		totalLen += float64(len(idx.tokens))
	}
	idf := make(map[string]float64, len(docFreq))
	docCount := float64(len(chunks))
	for token, df := range docFreq {
		idf[token] = math.Log((docCount-df+0.5)/(df+0.5) + 1)
	}
	avgLen := totalLen / math.Max(docCount, 1)
	return idf, avgLen
}

func (c *chunkIndex) cloneChunk() *types.Chunk {
	cloned := *c.chunk
	if c.chunk.Metadata != nil {
		cloned.Metadata = make(map[string]any, len(c.chunk.Metadata))
		for k, v := range c.chunk.Metadata {
			cloned.Metadata[k] = v
		}
	}
	return &cloned
}
