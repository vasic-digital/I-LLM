package client

import (
	"context"
	"errors"
	"testing"

	"digital.vasic.illm/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rspTestRunner is a deterministic unit-test stand-in for a real LLM Runner.
// CONST-050(A) permits mocks/stubs in unit tests only — production code MUST
// receive a real LLM-dispatching Runner via SetRunner, otherwise New()'s
// default returns ErrBaselineRunnerNotConfigured (round-23 §11.4 audit fix).
// Returns "RSP:" + prompt (truncated to 128 chars) so tests have a signal.
func rspTestRunner(_ context.Context, prompt string) (string, error) {
	limit := len(prompt)
	if limit > 128 {
		limit = 128
	}
	return "RSP:" + prompt[:limit], nil
}

// newTestClient builds a client with the RSP stub installed so unit tests
// have deterministic behaviour without depending on a real LLM provider.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New()
	require.NoError(t, err)
	c.SetRunner(rspTestRunner)
	return c
}

func TestNew(t *testing.T) {
	client, err := New()
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NoError(t, client.Close())
}

func TestDoubleClose(t *testing.T) {
	client, err := New()
	require.NoError(t, err)
	assert.NoError(t, client.Close())
	assert.NoError(t, client.Close())
}

func TestConfig(t *testing.T) {
	client, err := New()
	require.NoError(t, err)
	defer client.Close()
	assert.NotNil(t, client.Config())
}

func TestGetPattern(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	p, err := c.GetPattern(context.Background(), "cot-basic")
	require.NoError(t, err)
	assert.Equal(t, "Chain-of-Thought", p.Name)

	_, err = c.GetPattern(context.Background(), "does-not-exist")
	assert.Error(t, err)
}

func TestListPatterns(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	all, err := c.ListPatterns(context.Background(), "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 3)

	reasoning, err := c.ListPatterns(context.Background(), "reasoning")
	require.NoError(t, err)
	for _, p := range reasoning {
		assert.Equal(t, "reasoning", p.Category)
	}
}

func TestRenderPattern(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	p, err := c.GetPattern(context.Background(), "few-shot")
	require.NoError(t, err)
	out, err := c.RenderPattern(context.Background(), *p, map[string]string{
		"examples": "1->2, 2->4",
		"task":     "3->?",
	})
	require.NoError(t, err)
	assert.Contains(t, out, "1->2, 2->4")
	assert.Contains(t, out, "3->?")
}

func TestCreateAgent(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	a, err := c.CreateAgent(context.Background(), types.AgentConfig{
		Name:  "Researcher",
		Model: "gpt-4",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, a.ID)
	assert.InDelta(t, 0.7, a.Config.Temperature, 1e-9)
}

func TestChainOfThought(t *testing.T) {
	c := newTestClient(t)
	defer c.Close()

	r, err := c.ChainOfThought(context.Background(), "17 * 23", "gpt-4")
	require.NoError(t, err)
	assert.True(t, r.Success)
	assert.NotEmpty(t, r.FinalOutput)
	assert.Len(t, r.Steps, 1)
}

// TestChainOfThoughtWithoutInjectedRunner_ReturnsSentinel asserts the
// round-23 §11.4 audit fix: New()'s default Runner returns
// ErrBaselineRunnerNotConfigured when SetRunner is not called, instead
// of the previous silent "RSP:" stub that produced fabricated reasoning.
func TestChainOfThoughtWithoutInjectedRunner_ReturnsSentinel(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	_, err = c.ChainOfThought(context.Background(), "17 * 23", "gpt-4")
	require.Error(t, err, "ChainOfThought without injected Runner MUST surface the sentinel error, not return fabricated data")
	require.True(t, errors.Is(err, ErrBaselineRunnerNotConfigured), "wrapped error MUST be ErrBaselineRunnerNotConfigured; got %v", err)
}

func TestTreeOfThought(t *testing.T) {
	c := newTestClient(t)
	defer c.Close()

	tr, err := c.TreeOfThought(context.Background(), "design a cache", "gpt-4", 4)
	require.NoError(t, err)
	assert.Equal(t, 4, tr.Breadth)
	assert.Len(t, tr.Branches, 4)
}

// TestTreeOfThoughtWithoutInjectedRunner_ReturnsSentinel — sentinel
// propagates through the ToT path (which calls ChainOfThought internally).
func TestTreeOfThoughtWithoutInjectedRunner_ReturnsSentinel(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	_, err = c.TreeOfThought(context.Background(), "design a cache", "gpt-4", 3)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBaselineRunnerNotConfigured))
}

func TestRunChain(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	// Register an echoing runner so we can assert output flow.
	c.SetRunner(func(_ context.Context, prompt string) (string, error) {
		return "OUT[" + prompt + "]", nil
	})

	chain := types.PromptChain{
		ID:          "test-chain",
		Name:        "test",
		Description: "desc",
		Steps: []types.ChainStep{
			{Name: "s1", PromptTemplate: "{{x}}", OutputKey: "a"},
			{Name: "s2", PromptTemplate: "use {{a}}", OutputKey: "b"},
		},
	}
	r, err := c.RunChain(context.Background(), chain, map[string]string{"x": "hi"})
	require.NoError(t, err)
	assert.Equal(t, 2, r.Iterations)
	assert.True(t, r.Success)
	assert.Contains(t, r.FinalOutput, "use OUT[hi]")
}

// TestRunChainWithoutInjectedRunner_ReturnsSentinel — sentinel propagates
// through the RunChain path (the first step's prompt is dispatched without
// SetRunner, surfacing ErrBaselineRunnerNotConfigured wrapped as Unavailable).
func TestRunChainWithoutInjectedRunner_ReturnsSentinel(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	chain := types.PromptChain{
		ID: "no-runner-chain", Name: "no-runner-chain", Description: "d",
		Steps: []types.ChainStep{{Name: "s1", PromptTemplate: "hi", OutputKey: "o"}},
	}
	_, err = c.RunChain(context.Background(), chain, nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrBaselineRunnerNotConfigured))
}

func TestGetCategories(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	cats, err := c.GetCategories(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, cats)
}

func TestRegisterPattern(t *testing.T) {
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	err = c.RegisterPattern(types.ConversationPattern{
		ID: "custom", Name: "Custom", Description: "desc",
		Category: "x", Template: "hello {{name}}",
	})
	require.NoError(t, err)

	p, err := c.GetPattern(context.Background(), "custom")
	require.NoError(t, err)
	assert.Equal(t, "Custom", p.Name)
}
