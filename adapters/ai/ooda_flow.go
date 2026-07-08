package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ooda"
	"github.com/duynguyendang/manglekit/sdk/ports"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type OODAFlowInput struct {
	Input    string          `json:"input"`
	Intent   string          `json:"intent,omitempty"`
	TaskType ooda.TaskType  `json:"task_type,omitempty"`
}

type OODAFlowOutput struct {
	Output         string                     `json:"output"`
	Status         ooda.VerifyStatus         `json:"status"`
	PhaseDurations map[ooda.Phase]time.Duration `json:"phase_durations"`
	RetryCount     int                       `json:"retry_count"`
	AuditTrail     string                    `json:"audit_trail,omitempty"`
	Error          string                    `json:"error,omitempty"`
}

type OODAFlowConfig struct {
	Memory          ooda.Memory
	Brain           ooda.Brain
	Executor        ooda.Executor
	Dispatcher      *ooda.Dispatcher
	KnowledgeStore  ports.KnowledgeStore
	TransientStore  ports.TransientStore
	MaxRetries      int
	Timeout         time.Duration
	GenerateOptions []core.GenerateOption
	// ParadoxThreshold is the EAST magnitude above which cognitive paradox
	// injection is triggered. Defaults to 0.8 when unset.
	ParadoxThreshold float64
}

type OODAFlow struct {
	config *OODAFlowConfig
}

func NewOODAFlow(cfg *OODAFlowConfig) *OODAFlow {
	if cfg == nil {
		cfg = &OODAFlowConfig{}
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.ParadoxThreshold <= 0 {
		cfg.ParadoxThreshold = 0.8
	}
	return &OODAFlow{config: cfg}
}

func (f *OODAFlow) Run(ctx context.Context, input *OODAFlowInput) (*OODAFlowOutput, error) {
	taskType := input.TaskType
	if taskType == "" {
		taskType = ooda.TaskTypeGeneration
	}
	intent := ooda.IntentStr(input.Intent)
	if intent == "" {
		intent = ooda.IntentStr("default")
	}

	frame := ooda.NewCognitiveFrame(input.Input, intent, taskType)
	frame.MaxRetries = f.config.MaxRetries
	frame.Timeout = f.config.Timeout
	frame.EAST.ParadoxThreshold = f.config.ParadoxThreshold

	if f.config.Memory != nil {
		frame.Memory = f.config.Memory
	}
	if f.config.Brain != nil {
		frame.Brain = f.config.Brain
	}
	if f.config.Executor != nil {
		frame.Executor = f.config.Executor
	}
	if f.config.Dispatcher != nil {
		frame.Dispatcher = f.config.Dispatcher
	}
	frame.KnowledgeStore = f.config.KnowledgeStore
	frame.TransientStore = f.config.TransientStore

	_, err := ooda.RunOODA(ctx, frame)
	if err != nil {
		return &OODAFlowOutput{
			Error:         err.Error(),
			Status:        frame.Status,
			PhaseDurations: frame.PhaseDurations,
			RetryCount:    frame.RetryCount,
		}, nil
	}

	output := &OODAFlowOutput{
		Status:         frame.Status,
		PhaseDurations: frame.PhaseDurations,
		RetryCount:     frame.RetryCount,
	}

	if runResult, ok := frame.ActionResult.(string); ok {
		output.Output = runResult
	} else if frame.RawContext != nil {
		if out, ok := frame.RawContext["output"].(string); ok {
			output.Output = out
		}
	}

	if frame.AuditTrail != nil {
		output.AuditTrail = frame.AuditTrail.Summary()
	}

	return output, nil
}

func (f *OODAFlow) DefineFlow(g *genkit.Genkit, name string) {
	genkit.DefineFlow(g, name, func(ctx context.Context, input *OODAFlowInput) (*OODAFlowOutput, error) {
		return f.Run(ctx, input)
	})
}

func (f *OODAFlow) DefineStreamingFlow(g *genkit.Genkit, name string) {
	genkit.DefineStreamingFlow(g, name,
		func(ctx context.Context, input *OODAFlowInput, sendChunk ai.ModelStreamCallback) (*OODAFlowOutput, error) {
			result, err := f.Run(ctx, input)
			if err != nil {
				return result, err
			}

			if result.Output != "" {
				sendChunk(ctx, &ai.ModelResponseChunk{
					Content: []*ai.Part{ai.NewTextPart(result.Output)},
				})
			}

			return result, nil
		},
	)
}

type OODAFlowWithMiddleware struct {
	*OODAFlow
	g      *genkit.Genkit
}

func NewOODAFlowWithGenkit(cfg *OODAFlowConfig, g *genkit.Genkit) *OODAFlowWithMiddleware {
	return &OODAFlowWithMiddleware{
		OODAFlow: NewOODAFlow(cfg),
		g:        g,
	}
}

func (f *OODAFlowWithMiddleware) Run(ctx context.Context, input *OODAFlowInput) (*OODAFlowOutput, error) {
	return f.OODAFlow.Run(ctx, input)
}

type OODAGenkitBridge struct {
	g *genkit.Genkit
	f *OODAFlow
}

func NewOODAGenkitBridge(g *genkit.Genkit, cfg *OODAFlowConfig) *OODAGenkitBridge {
	return &OODAGenkitBridge{
		g: g,
		f: NewOODAFlow(cfg),
	}
}

func (b *OODAGenkitBridge) DefineOODAFlow(name string) {
	b.f.DefineFlow(b.g, name)
}

func (b *OODAGenkitBridge) DefineOODAStreamingFlow(name string) {
	b.f.DefineStreamingFlow(b.g, name)
}

func (b *OODAGenkitBridge) RunOODA(ctx context.Context, input *OODAFlowInput) (*OODAFlowOutput, error) {
	return b.f.Run(ctx, input)
}

type OODAExecutorConfig struct {
	MaxRetries     int
	Timeout        time.Duration
	Middleware     []ai.Middleware
	KnowledgeStore ports.KnowledgeStore
	TransientStore ports.TransientStore
}

type OODAExecutor struct {
	config *OODAExecutorConfig
}

func NewOODAExecutor(cfg *OODAExecutorConfig) *OODAExecutor {
	return &OODAExecutor{config: cfg}
}

func (e *OODAExecutor) Execute(ctx context.Context, input *OODAFlowInput) (*OODAFlowOutput, error) {
	cfg := &OODAFlowConfig{
		MaxRetries:     e.config.MaxRetries,
		Timeout:        e.config.Timeout,
		KnowledgeStore: e.config.KnowledgeStore,
		TransientStore: e.config.TransientStore,
	}

	flow := NewOODAFlow(cfg)
	return flow.Run(ctx, input)
}

func (e *OODAExecutor) WithMaxRetries(maxRetries int) *OODAExecutor {
	e.config.MaxRetries = maxRetries
	return e
}

func (e *OODAExecutor) WithTimeout(timeout time.Duration) *OODAExecutor {
	e.config.Timeout = timeout
	return e
}

func (e *OODAExecutor) WithMiddleware(mw ...ai.Middleware) *OODAExecutor {
	e.config.Middleware = append(e.config.Middleware, mw...)
	return e
}

type FlowRegistry struct {
	flows map[string]*OODAFlow
	g     *genkit.Genkit
}

func NewFlowRegistry(g *genkit.Genkit) *FlowRegistry {
	return &FlowRegistry{
		flows: make(map[string]*OODAFlow),
		g:     g,
	}
}

func (r *FlowRegistry) Register(name string, flow *OODAFlow) {
	r.flows[name] = flow
}

func (r *FlowRegistry) RegisterAndDefine(name string, flow *OODAFlow) {
	r.flows[name] = flow
	flow.DefineFlow(r.g, name)
}

func (r *FlowRegistry) Get(name string) (*OODAFlow, bool) {
	f, ok := r.flows[name]
	return f, ok
}

func (r *FlowRegistry) Run(ctx context.Context, name string, input *OODAFlowInput) (*OODAFlowOutput, error) {
	f, ok := r.flows[name]
	if !ok {
		return nil, fmt.Errorf("flow %s not found", name)
	}
	return f.Run(ctx, input)
}
