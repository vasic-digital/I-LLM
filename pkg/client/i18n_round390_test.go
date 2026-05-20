package client

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.illm/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// round390ClientRecorder captures keys requested by the client package so
// the paired-mutation test can prove every migrated user-facing literal in
// client.go funnels through types.Tr(). CONST-050(A): unit-test-only stub.
type round390ClientRecorder struct {
	calls map[string]int
}

func (r *round390ClientRecorder) Translate(key string, _ map[string]interface{}) string {
	r.calls[key]++
	return "XX:" + key
}

// TestRound390Client_SeedDefaultsRouteThroughTr asserts the built-in
// pattern/chain catalogue Names and Descriptions resolve via the i18n seam.
// With a recording Translator installed BEFORE New(), every seeded Name/Desc
// must carry the "XX:" stamp — a surviving hardcoded literal would not.
func TestRound390Client_SeedDefaultsRouteThroughTr(t *testing.T) {
	rec := &round390ClientRecorder{calls: map[string]int{}}
	types.SetTranslator(rec)
	defer types.SetTranslator(nil)

	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	patterns, err := c.ListPatterns(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, patterns)
	for _, p := range patterns {
		assert.Truef(t, strings.HasPrefix(p.Name, "XX:"),
			"pattern %q Name not routed through Tr(): %q", p.ID, p.Name)
		assert.Truef(t, strings.HasPrefix(p.Description, "XX:"),
			"pattern %q Description not routed through Tr(): %q", p.ID, p.Description)
	}

	// Every migrated seed key must have been observed.
	for _, key := range []string{
		"client.pattern.cot.name", "client.pattern.cot.desc",
		"client.pattern.react.name", "client.pattern.react.desc",
		"client.pattern.fewshot.name", "client.pattern.fewshot.desc",
		"client.chain.summtrans.name", "client.chain.summtrans.desc",
	} {
		assert.Positivef(t, rec.calls[key], "seed key %q never routed through Tr()", key)
	}
}

// TestRound390Client_ErrorMessagesRouteThroughTr asserts client.go error
// strings funnel through the i18n seam. A non-English Translator makes a
// surviving hardcoded English literal observable.
func TestRound390Client_ErrorMessagesRouteThroughTr(t *testing.T) {
	types.SetTranslator(round390SrClientTranslator{})
	defer types.SetTranslator(nil)

	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	ctx := context.Background()

	// pattern not found
	_, err = c.GetPattern(ctx, "does-not-exist-round390")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sr:client.err.pattern_not_found")

	// invalid pattern (empty -> Validate fails -> wrapped invalid_pattern)
	err = c.RegisterPattern(types.ConversationPattern{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sr:client.err.invalid_pattern")

	// problem required
	_, err = c.ChainOfThought(ctx, "", "model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sr:client.err.problem_required")

	// invalid agent config
	_, err = c.CreateAgent(ctx, types.AgentConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sr:client.err.invalid_agent_config")
}

// TestRound390Client_DefaultNoopKeepsEnglish confirms the library remains
// fully usable with no i18n backend wired (NoopTranslator English fallback).
func TestRound390Client_DefaultNoopKeepsEnglish(t *testing.T) {
	types.SetTranslator(nil)
	c, err := New()
	require.NoError(t, err)
	defer c.Close()

	patterns, err := c.ListPatterns(context.Background(), "")
	require.NoError(t, err)
	var sawCoT bool
	for _, p := range patterns {
		if p.ID == "cot-basic" {
			sawCoT = true
			assert.Equal(t, "Chain-of-Thought", p.Name)
			assert.Equal(t, "Basic step-by-step reasoning scaffold.", p.Description)
		}
	}
	assert.True(t, sawCoT, "default cot-basic pattern missing")
}

type round390SrClientTranslator struct{}

func (round390SrClientTranslator) Translate(key string, _ map[string]interface{}) string {
	return "sr:" + key
}
