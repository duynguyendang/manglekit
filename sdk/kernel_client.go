package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/kernel"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/multiagent"
)

// KernelClient is the primary entry point for the firmware-ized manglekit
// It provides zero-config OODA execution with profile-based loading from Nexus
type KernelClient struct {
	Client       *Client
	kernelLoader *kernel.Loader
	agentSystem  *multiagent.AgentSystem
	oodaLoop     *DefaultOODALoop
}

// NewKernelClient creates a kernel client with embedded rules
func NewKernelClient(ctx context.Context, opts ...KernelOption) (*KernelClient, error) {
	kc := &KernelClient{
		Client: &Client{
			logger:      logger.NewDefault(),
			agentMemory: NewHybridMemory(core.NopStore{}, core.NopVectorStore{}, core.NopEmbedder{}),
			registry:    make(map[string]core.Action),
			failureMode: FailModeClosed,
		},
	}

	var err error
	kc.kernelLoader, err = kernel.NewLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kernel loader: %w", err)
	}

	if err := ensureDependencies(kc.Client); err != nil {
		return nil, fmt.Errorf("failed to initialize client dependencies: %w", err)
	}

	if err := kc.Client.engine.LoadPolicy(ctx, kc.kernelLoader.GetKernel()); err != nil {
		return nil, fmt.Errorf("failed to load embedded kernel: %w", err)
	}

	kc.agentSystem, err = multiagent.NewAgentSystem(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent system: %w", err)
	}

	if err := kc.agentSystem.LoadAgentDefinitions(ctx); err != nil {
		kc.Client.logger.Warn("failed to load agent definitions", "error", err)
	}

	kc.oodaLoop = &DefaultOODALoop{
		brain:    &brainAdapter{kc.Client.engine},
		executor: &defaultExecutor{client: kc.Client},
		memory:   kc.Client.agentMemory,
		logger:   kc.Client.logger,
	}

	for _, opt := range opts {
		if err := opt(kc); err != nil {
			return nil, err
		}
	}

	if err := ensureDependencies(kc.Client); err != nil {
		return nil, err
	}

	return kc, nil
}

// LoadProfile loads a Nexus security/business profile
func (kc *KernelClient) LoadProfile(profile kernel.Profile) error {
	ctx := context.Background()
	rules, err := kc.kernelLoader.Merge(profile)
	if err != nil {
		return fmt.Errorf("failed to merge profile: %w", err)
	}

	if err := kc.Client.engine.LoadPolicy(ctx, rules); err != nil {
		return fmt.Errorf("failed to load profile rules: %w", err)
	}

	kc.Client.logger.Info("loaded profile", "name", profile.Name)
	return nil
}

// Run executes action with zero-config OODA
func (kc *KernelClient) Run(ctx context.Context, actionName string, input interface{}) (interface{}, error) {
	kc.Client.registryLock.RLock()
	action, ok := kc.Client.registry[actionName]
	kc.Client.registryLock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("action not found: %s", actionName)
	}

	envelope := core.Envelope{Payload: input}
	frame := &CognitiveFrame{
		Input:      fmt.Sprintf("%v", input),
		Timeout:    60 * 1e9,
		Memory:     kc.Client.agentMemory,
		Brain:      kc.oodaLoop.brain,
		executor:   kc.oodaLoop.executor,
		MaxRetries: 3,
	}

	if err := kc.oodaLoop.Run(ctx, frame, action, envelope); err != nil {
		return nil, fmt.Errorf("ooda failed: %w", err)
	}

	return frame.ActionResult, nil
}

// RunWithProfile executes action with a specific profile
func (kc *KernelClient) RunWithProfile(ctx context.Context, profile kernel.Profile, actionName string, input interface{}) (interface{}, error) {
	if err := kc.LoadProfile(profile); err != nil {
		return nil, err
	}
	return kc.Run(ctx, actionName, input)
}

// DefaultOODALoop provides zero-config OODA execution
type DefaultOODALoop struct {
	brain    Brain
	executor Executor
	memory   core.AgentMemory
	logger   core.Logger
}

// Run executes the OODA loop: Observe → Orient → Decide → Act → Verify
func (loop *DefaultOODALoop) Run(ctx context.Context, frame *CognitiveFrame, action core.Action, input core.Envelope) error {
	frame.Phase = PhaseObserve
	frame.Context = append(frame.Context, Atom{
		Subject:   "input",
		Predicate: "raw",
		Object:    fmt.Sprintf("%v", input.Payload),
		Weight:    1.0,
	})

	frame.Phase = PhaseOrient
	if loop.memory != nil {
		if recallResult, err := loop.memory.Recall(ctx, fmt.Sprintf("%v", input.Payload)); err == nil {
			frame.Context = append(frame.Context, Atom{
				Subject:   "memory",
				Predicate: "recall",
				Object:    recallResult,
				Weight:    1.0,
			})
		}
	}

	frame.Phase = PhaseDecide
	decision, err := loop.brain.Evaluate(ctx, frame)
	if err != nil {
		return fmt.Errorf("decide phase failed: %w", err)
	}
	frame.Decision = decision

	frame.Phase = PhaseAct
	result, err := loop.executor.Execute(ctx, frame, decision)
	if err != nil {
		return fmt.Errorf("act phase failed: %w", err)
	}
	frame.ActionResult = result

	frame.Phase = PhaseVerify

	return nil
}

// CognitiveFrame holds execution context
type CognitiveFrame struct {
	Input          string
	Timeout        int64
	Memory         core.AgentMemory
	Brain          Brain
	executor       Executor
	Phase          Phase
	Context        []Atom
	Decision       core.Decision
	ActionResult   interface{}
	RetryCount     int
	MaxRetries     int
	AuditTrail     *core.AuditTrail
	KnowledgeStore KnowledgeStore
	TransientStore TransientStore
	SessionID      string
	WorkflowID     string
	PhaseDurations map[Phase]int64
}

// Phase represents OODA phase
type Phase int

const (
	PhaseObserve Phase = iota
	PhaseOrient
	PhaseDecide
	PhaseAct
	PhaseVerify
	PhaseRefine
)

// Atom represents a cognitive fact
type Atom struct {
	Subject   string
	Predicate string
	Object    string
	Weight    float64
}

// Brain evaluates policy
type Brain interface {
	Evaluate(ctx context.Context, frame *CognitiveFrame) (core.Decision, error)
	LoadPolicy(ctx context.Context, rules string) error
}

// Executor runs actions
type Executor interface {
	Execute(ctx context.Context, frame *CognitiveFrame, decision core.Decision) (interface{}, error)
}

// KnowledgeStore for long-term memory
type KnowledgeStore interface {
	Recall(ctx context.Context, query string, k int, graphID string) ([]Atom, error)
}

// TransientStore for session memory
type TransientStore interface {
	ToAtoms(ctx context.Context, sessionID string) ([]Atom, error)
	Put(ctx context.Context, sessionID, key string, fact *TransientFact) error
}

// Decision is the result type from policy evaluation
type Decision = core.Decision

// brainAdapter wraps core.Evaluator
type brainAdapter struct {
	engine core.Evaluator
}

func (b *brainAdapter) Evaluate(ctx context.Context, frame *CognitiveFrame) (core.Decision, error) {
	envelope := core.Envelope{Payload: frame.Input}
	return b.engine.AssessPlan(ctx, envelope)
}

func (b *brainAdapter) LoadPolicy(ctx context.Context, rules string) error {
	return b.engine.LoadPolicy(ctx, rules)
}

// defaultExecutor routes to registered actions
type defaultExecutor struct {
	client *Client
}

func (e *defaultExecutor) Execute(ctx context.Context, frame *CognitiveFrame, decision core.Decision) (interface{}, error) {
	if decision.Action == nil {
		return nil, fmt.Errorf("no decision action")
	}

	e.client.registryLock.RLock()
	action, ok := e.client.registry[decision.Action.Name]
	e.client.registryLock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("action not registered: %s", decision.Action.Name)
	}

	return action.Execute(ctx, core.Envelope{Payload: frame.Input})
}

// SecurityProfileRegistry for Nexus profiles
type SecurityProfileRegistry interface {
	GetProfile(name string) (kernel.Profile, error)
	ListProfiles() []string
}

// DefaultProfileRegistry provides built-in profiles
type DefaultProfileRegistry struct {
	profiles map[string]kernel.Profile
}

// NewDefaultProfileRegistry creates registry with default profiles
func NewDefaultProfileRegistry() *DefaultProfileRegistry {
	return &DefaultProfileRegistry{
		profiles: map[string]kernel.Profile{
			"default": {
				Name:     "default",
				Rules:    `allow(_, _, _) :- true.`,
				Metadata: map[string]string{"description": "Permissive default"},
			},
			"strict": {
				Name: "strict",
				Rules: `
validation_severity(Rule, "error") :- validation_rule(Rule, _).
validation_severity("output_not_empty", "critical").
`,
				Metadata: map[string]string{"description": "Strict validation"},
			},
		},
	}
}

func (r *DefaultProfileRegistry) GetProfile(name string) (kernel.Profile, error) {
	p, ok := r.profiles[name]
	if !ok {
		return kernel.Profile{}, fmt.Errorf("profile not found: %s", name)
	}
	return p, nil
}

func (r *DefaultProfileRegistry) ListProfiles() []string {
	names := make([]string, 0, len(r.profiles))
	for n := range r.profiles {
		names = append(names, n)
	}
	return names
}

// KernelOption configures KernelClient
type KernelOption func(*KernelClient) error

// WithProfile loads a profile
func WithProfile(profile kernel.Profile) KernelOption {
	return func(kc *KernelClient) error {
		return kc.LoadProfile(profile)
	}
}

// WithProfileName loads a named profile from registry
func WithProfileName(name string, registry SecurityProfileRegistry) KernelOption {
	return func(kc *KernelClient) error {
		p, err := registry.GetProfile(name)
		if err != nil {
			return err
		}
		return kc.LoadProfile(p)
	}
}

// TransientFact represents a session fact
type TransientFact struct {
	Subject   string
	Predicate string
	Object    string
	Graph     string
}
