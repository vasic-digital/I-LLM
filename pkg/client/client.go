// Package client provides the Go client for the I-LLM library.
//
// I-LLM provides interactive LLM conversation patterns (few-shot, ReAct,
// chain-of-thought, tree-of-thought) as a reusable template library with
// a simple `{{variable}}` substitution renderer. The client ships with a
// small set of built-in patterns and chains so it is immediately usable;
// callers can register their own.
//
// Basic usage:
//
//	import illm "digital.vasic.illm/pkg/client"
//
//	c, err := illm.New()
//	if err != nil { log.Fatal(err) }
//	defer c.Close()
package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"digital.vasic.pliniuscommon/pkg/config"
	"digital.vasic.pliniuscommon/pkg/errors"

	. "digital.vasic.illm/pkg/types"
)

// ToolHandler executes a tool invocation in a ReAct loop.
type ToolHandler func(ctx context.Context, name, input string) (string, error)

// Runner generates a completion for a prompt (used by ChainOfThought /
// TreeOfThought / RunChain).
type Runner func(ctx context.Context, prompt string) (string, error)

// Client is the Go client for I-LLM.
type Client struct {
	cfg    *config.Config
	mu     sync.RWMutex
	closed bool

	patterns map[string]ConversationPattern
	chains   map[string]PromptChain
	tools    map[string]ToolHandler
	runner   Runner
}

// New creates a new I-LLM client with default patterns and chains seeded.
func New(opts ...config.Option) (*Client, error) {
	cfg := config.New("i-llm", opts...)
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidArgument, "i-llm",
			"invalid configuration", err)
	}
	c := &Client{
		cfg:      cfg,
		patterns: make(map[string]ConversationPattern),
		chains:   make(map[string]PromptChain),
		tools:    make(map[string]ToolHandler),
		runner:   baselineRunner,
	}
	c.seedDefaults()
	return c, nil
}

// NewFromConfig creates a client from a config object.
func NewFromConfig(cfg *config.Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidArgument, "i-llm",
			"invalid configuration", err)
	}
	c := &Client{
		cfg:      cfg,
		patterns: make(map[string]ConversationPattern),
		chains:   make(map[string]PromptChain),
		tools:    make(map[string]ToolHandler),
		runner:   baselineRunner,
	}
	c.seedDefaults()
	return c, nil
}

// Close gracefully closes the client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return nil
}

// Config returns the client configuration.
func (c *Client) Config() *config.Config { return c.cfg }

// SetRunner injects the LLM runner used by ChainOfThought, TreeOfThought, RunChain.
func (c *Client) SetRunner(r Runner) {
	if r == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.runner = r
}

// RegisterPattern adds or overrides a conversation pattern.
func (c *Client) RegisterPattern(p ConversationPattern) error {
	if err := p.Validate(); err != nil {
		return errors.Wrap(errors.ErrCodeInvalidArgument, "i-llm",
			"invalid pattern", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.patterns[p.ID] = p
	return nil
}

// RegisterChain adds or overrides a prompt chain.
func (c *Client) RegisterChain(p PromptChain) error {
	if err := p.Validate(); err != nil {
		return errors.Wrap(errors.ErrCodeInvalidArgument, "i-llm",
			"invalid chain", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chains[p.ID] = p
	return nil
}

// RegisterTool wires in a ReAct-style tool handler.
func (c *Client) RegisterTool(name string, h ToolHandler) {
	if name == "" || h == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tools[name] = h
}

// GetPattern retrieves a registered pattern by id.
func (c *Client) GetPattern(ctx context.Context, id string) (*ConversationPattern, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if p, ok := c.patterns[id]; ok {
		out := p
		return &out, nil
	}
	return nil, errors.New(errors.ErrCodeNotFound, "i-llm", "pattern not found")
}

// ListPatterns returns all patterns (optionally filtered by category).
func (c *Client) ListPatterns(ctx context.Context, category string) ([]ConversationPattern, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ConversationPattern, 0, len(c.patterns))
	for _, p := range c.patterns {
		if category == "" || strings.EqualFold(p.Category, category) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// RenderPattern substitutes `{{var}}` occurrences in a pattern template.
func (c *Client) RenderPattern(_ context.Context, pattern ConversationPattern, vars map[string]string) (string, error) {
	return renderTemplate(pattern.Template, vars), nil
}

// CreateAgent creates a ReAct agent instance.
func (c *Client) CreateAgent(_ context.Context, cfg AgentConfig) (*Agent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidArgument, "i-llm",
			"invalid agent config", err)
	}
	cfg.Defaults()
	return &Agent{Config: cfg, ID: fmt.Sprintf("agent-%s", strings.ToLower(cfg.Name))}, nil
}

// RunChain executes a registered chain (or an inline chain) step by step.
// Each step's rendered prompt is sent through the runner and stored under
// its OutputKey (or the step name) in the working set of variables.
func (c *Client) RunChain(ctx context.Context, chain PromptChain, inputs map[string]string) (*ChainResult, error) {
	if err := chain.Validate(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidArgument, "i-llm",
			"invalid chain", err)
	}
	c.mu.RLock()
	runner := c.runner
	c.mu.RUnlock()

	vars := make(map[string]string, len(inputs))
	for k, v := range inputs {
		vars[k] = v
	}

	res := &ChainResult{Success: true}
	for _, step := range chain.Steps {
		prompt := renderTemplate(step.PromptTemplate, vars)
		out, err := runner(ctx, prompt)
		if err != nil {
			res.Success = false
			return res, errors.Wrap(errors.ErrCodeUnavailable, "i-llm",
				"runner failed", err)
		}
		key := step.OutputKey
		if key == "" {
			key = step.Name
		}
		vars[key] = out
		res.Iterations++
		res.FinalOutput = out
		res.TokenUsage += len(prompt) + len(out)
	}
	return res, nil
}

// ChainOfThought prompts the runner with a CoT scaffold.
func (c *Client) ChainOfThought(ctx context.Context, problem string, model string) (*ChainResult, error) {
	if problem == "" {
		return nil, errors.New(errors.ErrCodeInvalidArgument, "i-llm", "problem is required")
	}
	c.mu.RLock()
	runner := c.runner
	c.mu.RUnlock()
	prompt := fmt.Sprintf("Problem: %s\n\nLet's think step by step.\n", problem)
	out, err := runner(ctx, prompt)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeUnavailable, "i-llm",
			"runner failed", err)
	}
	return &ChainResult{
		FinalOutput: out,
		Iterations:  1,
		TokenUsage:  len(prompt) + len(out),
		Success:     true,
		Steps: []ReActStep{{
			StepNum:     1,
			Thought:     "Chain-of-thought decomposition",
			Action:      "reason",
			ActionInput: problem,
			Observation: out,
		}},
	}, nil
}

// TreeOfThought explores `breadth` parallel reasoning branches and aggregates them.
func (c *Client) TreeOfThought(ctx context.Context, problem string, model string, breadth int) (*TreeResult, error) {
	if problem == "" {
		return nil, errors.New(errors.ErrCodeInvalidArgument, "i-llm", "problem is required")
	}
	if breadth <= 0 {
		breadth = 3
	}
	tr := &TreeResult{Breadth: breadth, Success: true}
	for i := 0; i < breadth; i++ {
		subProblem := fmt.Sprintf("%s (branch %d)", problem, i+1)
		cr, err := c.ChainOfThought(ctx, subProblem, model)
		if err != nil {
			tr.Success = false
			return tr, err
		}
		tr.Branches = append(tr.Branches, *cr)
		tr.TokenUsage += cr.TokenUsage
	}
	// aggregate by picking the longest output (baseline heuristic).
	if len(tr.Branches) > 0 {
		bestIdx := 0
		for i, b := range tr.Branches {
			if len(b.FinalOutput) > len(tr.Branches[bestIdx].FinalOutput) {
				bestIdx = i
			}
		}
		tr.FinalOutput = tr.Branches[bestIdx].FinalOutput
	}
	return tr, nil
}

// GetCategories lists unique pattern categories.
func (c *Client) GetCategories(_ context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, p := range c.patterns {
		seen[p.Category] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// --- internals ---

func (c *Client) seedDefaults() {
	patterns := []ConversationPattern{
		{
			ID: "cot-basic", Name: "Chain-of-Thought", Category: "reasoning",
			Description: "Basic step-by-step reasoning scaffold.",
			Template:    "Problem: {{problem}}\n\nLet's think step by step.",
			Variables:   []string{"problem"},
			Example:     "Problem: 17 * 23\n\nLet's think step by step.",
		},
		{
			ID: "react-basic", Name: "ReAct", Category: "agent",
			Description: "ReAct agent scaffold with Thought/Action/Observation.",
			Template: "Question: {{question}}\n\nThought:\nAction:\nAction Input:\n" +
				"Observation:\n...\nFinal Answer:",
			Variables: []string{"question"},
		},
		{
			ID: "few-shot", Name: "Few-Shot", Category: "prompt",
			Description: "Few-shot example scaffold.",
			Template:    "Examples:\n{{examples}}\n\nTask: {{task}}",
			Variables:   []string{"examples", "task"},
		},
	}
	for _, p := range patterns {
		c.patterns[p.ID] = p
	}

	chains := []PromptChain{
		{
			ID: "summarise-then-translate", Name: "Summarise then Translate",
			Category:    "demo",
			Description: "Two-step chain: summarise input, then translate the summary.",
			Steps: []ChainStep{
				{Name: "summarise", PromptTemplate: "Summarise briefly: {{text}}", OutputKey: "summary"},
				{Name: "translate", PromptTemplate: "Translate to {{lang}}: {{summary}}", OutputKey: "translated"},
			},
		},
	}
	for _, ch := range chains {
		c.chains[ch.ID] = ch
	}
}

func renderTemplate(tpl string, vars map[string]string) string {
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func baselineRunner(_ context.Context, prompt string) (string, error) {
	// Deterministic stand-in: prefix "RSP:" + prompt (truncated).
	limit := len(prompt)
	if limit > 128 {
		limit = 128
	}
	return "RSP:" + prompt[:limit], nil
}
