package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
)

// LLMStage is responsible for synthesizing a final answer by calling a large
// language model with the evidence gathered in previous stages.
type LLMStage struct {
	LLM       llm.Client
	MaxTokens int
	Logger    core.Logger
	Meter     core.Meter
}

// Name returns the identifier for the stage.
func (s *LLMStage) Name() string {
	return "llm"
}

// Execute prepares the LLM request, calls the LLM, and populates the context
// with the final response text and token usage metrics. It also generates
// the final citations for the answer.
func (s *LLMStage) Execute(p *PipelineContext) error {
	if s.LLM == nil {
		s.Logger.Warnf("LLM client is nil, skipping llm stage")
		return fmt.Errorf("LLM client is not configured")
	}

	// 1. Prepare passages and citations for the LLM.
	passages, err := s.prepareLlmRequest(p)
	if err != nil {
		s.Logger.Errorf("prepare llm request failed", "error", err)
		return fmt.Errorf("prepare llm request failed: %w", err)
	}

	// 2. Build the prompt data, including history.
	promptData := map[string]interface{}{
		"query":     p.Query.Text,
		"documents": passages,
	}
	if p.Query.Meta != nil {
		for k, v := range p.Query.Meta {
			promptData[k] = v
		}
	}
	if p.History != nil {
		promptData["history"] = p.History.Messages
	}

	// 3. Call the LLM.
	start := time.Now()
	llmRes, err := s.LLM.Complete(p.Ctx, llm.Request{
		Prompt:    fmt.Sprintf("%s context: %s", p.Query.Text, strings.Join(passages, " ")),
		Context:   passages,
		MaxTokens: s.MaxTokens,
		Data:      promptData,
	})
	if err != nil {
		s.Logger.Errorf("llm failed", "error", err)
		return fmt.Errorf("llm failed: %w", err)
	}
	p.LLMMS = float64(time.Since(start).Milliseconds())
	if s.Meter != nil {
		s.Meter.Record("manglekit.llm_ms", p.LLMMS)
	}

	// 4. Populate the context with the final response and metadata.
	p.Response = llmRes.Text
	p.Answer.Text = llmRes.Text
	if p.Answer.Meta == nil {
		p.Answer.Meta = make(map[string]any)
	}
	p.Answer.Meta["token_usage"] = llmRes.Usage

	return nil
}

// prepareLlmRequest generates citations from the documents in the context and
// returns a simple slice of document text (passages) for the LLM prompt.
func (s *LLMStage) prepareLlmRequest(p *PipelineContext) ([]string, error) {
	passages := make([]string, len(p.FinalDocs))
	for i, d := range p.FinalDocs {
		passages[i] = d.Text
	}

	// If we have reranked docs, use them to create citations with scores.
	if len(p.RerankedDocs) > 0 {
		p.Citations = make([]core.Citation, len(p.RerankedDocs))
		for i, rd := range p.RerankedDocs {
			p.Citations[i] = core.Citation{
				ID:      rd.Doc.ID,
				Source:  rd.Doc.Source,
				URI:     rd.Doc.URI,
				Snippet: rd.Doc.Text,
				Score:   rd.Score,
			}
		}
	} else { // Otherwise, create citations from the original documents without scores.
		p.Citations = make([]core.Citation, len(p.FinalDocs))
		for i, d := range p.FinalDocs {
			p.Citations[i] = core.Citation{
				ID:      d.ID,
				Source:  d.Source,
				URI:     d.URI,
				Snippet: d.Text,
			}
		}
	}
	p.Answer.Citations = p.Citations
	return passages, nil
}
