package client

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	"digital.vasic.illm/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderPatternWithMissingVars — unknown variables remain as `{{var}}` placeholders.
func TestRenderPatternWithMissingVars(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	p, err := c.GetPattern(context.Background(), "few-shot")
	require.NoError(t, err)

	rendered, err := c.RenderPattern(context.Background(), *p, map[string]string{"task": "T"})
	require.NoError(t, err)
	// examples not supplied → placeholder remains.
	assert.Contains(t, rendered, "{{examples}}")
	assert.Contains(t, rendered, "Task: T")
}

// TestRenderPatternAllVars — all placeholders substituted.
func TestRenderPatternAllVars(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	p, err := c.GetPattern(context.Background(), "cot-basic")
	require.NoError(t, err)
	rendered, err := c.RenderPattern(context.Background(), *p, map[string]string{"problem": "X"})
	require.NoError(t, err)
	assert.NotContains(t, rendered, "{{")
	assert.Contains(t, rendered, "Problem: X")
}

// TestChainOfThoughtEmptyProblem — empty problem errors.
func TestChainOfThoughtEmptyProblem(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	_, err = c.ChainOfThought(context.Background(), "", "m")
	assert.Error(t, err)
}

// TestChainOfThoughtRunnerError — runner error wraps Unavailable.
func TestChainOfThoughtRunnerError(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	c.SetRunner(func(_ context.Context, _ string) (string, error) { return "", stderrors.New("boom") })
	_, err = c.ChainOfThought(context.Background(), "p", "m")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner failed")
}

// TestTreeOfThoughtZeroBreadth — 0 breadth coerces to 3 per impl.
func TestTreeOfThoughtZeroBreadth(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	tr, err := c.TreeOfThought(context.Background(), "p", "m", 0)
	require.NoError(t, err)
	assert.Equal(t, 3, tr.Breadth)
	assert.Len(t, tr.Branches, 3)
}

// TestRegisterToolIdempotence — registering same name twice replaces handler.
func TestRegisterToolIdempotence(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	c.RegisterTool("t1", func(ctx context.Context, n, in string) (string, error) { return "first", nil })
	c.RegisterTool("t1", func(ctx context.Context, n, in string) (string, error) { return "second", nil })

	c.mu.RLock()
	h := c.tools["t1"]
	c.mu.RUnlock()
	out, err := h(context.Background(), "t1", "x")
	require.NoError(t, err)
	assert.Equal(t, "second", out)
}

// TestRegisterToolNilOrEmptyIgnored.
func TestRegisterToolNilOrEmptyIgnored(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	c.RegisterTool("", func(_ context.Context, _, _ string) (string, error) { return "", nil })
	c.RegisterTool("only-name", nil)

	c.mu.RLock()
	defer c.mu.RUnlock()
	_, emptyOk := c.tools[""]
	_, nameOk := c.tools["only-name"]
	assert.False(t, emptyOk)
	assert.False(t, nameOk)
}

// TestRegisterPatternValidation — invalid pattern rejected.
func TestRegisterPatternValidation(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	err = c.RegisterPattern(types.ConversationPattern{})
	assert.Error(t, err)
}

// TestRunChainInvalidChain — missing ID fails.
func TestRunChainInvalidChain(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	_, err = c.RunChain(context.Background(), types.PromptChain{}, nil)
	assert.Error(t, err)
}

// TestRunChainRunnerErrorPropagates.
func TestRunChainRunnerErrorPropagates(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	c.SetRunner(func(_ context.Context, _ string) (string, error) { return "", stderrors.New("fail") })

	chain := types.PromptChain{
		ID: "c1", Name: "c1", Description: "d",
		Steps: []types.ChainStep{{Name: "s1", PromptTemplate: "hello", OutputKey: "o1"}},
	}
	_, err = c.RunChain(context.Background(), chain, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner failed")
}

// TestRunChainVariableCarryThrough — step N output flows into step N+1 prompt.
func TestRunChainVariableCarryThrough(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	captured := make([]string, 0, 2)
	c.SetRunner(func(_ context.Context, prompt string) (string, error) {
		captured = append(captured, prompt)
		return "OUT-" + prompt[len(prompt)-1:], nil
	})
	chain := types.PromptChain{
		ID: "c1", Name: "c1", Description: "d",
		Steps: []types.ChainStep{
			{Name: "s1", PromptTemplate: "A", OutputKey: "x"},
			{Name: "s2", PromptTemplate: "B-{{x}}", OutputKey: "y"},
		},
	}
	res, err := c.RunChain(context.Background(), chain, nil)
	require.NoError(t, err)
	require.Len(t, captured, 2)
	assert.Equal(t, "A", captured[0])
	assert.True(t, strings.HasPrefix(captured[1], "B-OUT-"))
	assert.Equal(t, 2, res.Iterations)
}

// TestListPatternsFilterByCategory.
func TestListPatternsFilterByCategory(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	out, err := c.ListPatterns(context.Background(), "reasoning")
	require.NoError(t, err)
	for _, p := range out {
		assert.True(t, strings.EqualFold(p.Category, "reasoning"))
	}
}

// TestCreateAgentValidation — missing fields rejected.
func TestCreateAgentValidation(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	_, err = c.CreateAgent(context.Background(), types.AgentConfig{})
	assert.Error(t, err)

	a, err := c.CreateAgent(context.Background(), types.AgentConfig{Name: "Bot", Model: "gpt"})
	require.NoError(t, err)
	assert.Equal(t, "agent-bot", a.ID)
	assert.InDelta(t, 0.7, a.Config.Temperature, 1e-9)
}
