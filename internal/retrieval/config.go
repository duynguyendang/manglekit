package retrieval

// Config captures corpus and hybrid retrieval configuration.
type Config struct {
	Corpus CorpusConfig `yaml:"corpus"`
	Hybrid HybridConfig `yaml:"hybrid"`
	Rerank RerankConfig `yaml:"rerank"`
}

// CorpusConfig controls document ingestion and chunking.
type CorpusConfig struct {
	Path         string `yaml:"path"`
	ChunkSize    int    `yaml:"chunkSize"`
	ChunkOverlap int    `yaml:"chunkOverlap"`
}

// HybridConfig controls lexical and dense retrieval behaviour.
type HybridConfig struct {
	BM25 struct {
		Must   int `yaml:"must"`
		Should int `yaml:"should"`
	} `yaml:"bm25"`
	Dense struct {
		TopK     int    `yaml:"topK"`
		Model    string `yaml:"model"`
		StoreDir string `yaml:"storeDir"`
	} `yaml:"dense"`
}

// RerankConfig configures the fine-stage reranker.
type RerankConfig struct {
	MRL struct {
		TopK       int   `yaml:"topK"`
		Dimensions []int `yaml:"dimensions"`
	} `yaml:"mrl"`
}
