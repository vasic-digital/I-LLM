package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// round390RecordingTranslator captures every key requested and returns a
// locale-stamped value so a test can prove the seam is actually consulted.
// CONST-050(A): mocks/stubs permitted in unit tests only — this file is a
// unit test (no integration build tag).
type round390RecordingTranslator struct {
	calls map[string]int
}

func newRound390RecordingTranslator() *round390RecordingTranslator {
	return &round390RecordingTranslator{calls: map[string]int{}}
}

func (r *round390RecordingTranslator) Translate(key string, args map[string]interface{}) string {
	r.calls[key]++
	return "XX:" + key
}

// TestRound390_NoopTranslator_KnownKeysResolve asserts every bundle key
// the round-390 migration introduced resolves to a non-empty, non-key
// string under the default NoopTranslator.
func TestRound390_NoopTranslator_KnownKeysResolve(t *testing.T) {
	SetTranslator(nil) // restore default
	defer SetTranslator(nil)

	for key := range enBundle {
		got := Tr(key, nil)
		require.NotEmpty(t, got, "key %q resolved empty", key)
		assert.NotEqual(t, key, got, "key %q echoed instead of resolving to bundle value", key)
		assert.Equal(t, enBundle[key], got, "key %q must resolve to its bundle value", key)
	}
}

// TestRound390_NoopTranslator_UnknownKeyEchoes asserts an unknown key
// echoes loudly (never silently swallowed — a silent swallow would be a
// §11.4 PASS-bluff at the i18n layer).
func TestRound390_NoopTranslator_UnknownKeyEchoes(t *testing.T) {
	SetTranslator(nil)
	defer SetTranslator(nil)
	const unknown = "types.does.not.exist.round390"
	assert.Equal(t, unknown, Tr(unknown, nil))
}

// TestRound390_SetTranslator_RoutesValidationMessages is the paired-mutation
// proof: it swaps in a recording Translator and asserts every migrated
// validation call site in types.go funnels through Tr() — if a literal were
// left hardcoded the recorder would never see its key and the test fails.
func TestRound390_SetTranslator_RoutesValidationMessages(t *testing.T) {
	rec := newRound390RecordingTranslator()
	SetTranslator(rec)
	defer SetTranslator(nil)

	// ConversationPattern: description, id, name keys.
	cp := &ConversationPattern{}
	err := cp.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.description_required", err.Error())

	cp.Description = "d"
	err = cp.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.id_required", err.Error())

	cp.ID = "id"
	err = cp.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.name_required", err.Error())

	// AgentConfig: model key.
	ac := &AgentConfig{}
	err = ac.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.model_required", err.Error())

	// Tool: description key.
	tl := &Tool{}
	err = tl.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.description_required", err.Error())

	// PromptChain: id key (description set so we reach id branch).
	pc := &PromptChain{Description: "d"}
	err = pc.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.id_required", err.Error())

	// ChainStep: name key.
	cs := &ChainStep{}
	err = cs.Validate()
	require.Error(t, err)
	assert.Equal(t, "XX:types.validation.name_required", err.Error())

	// Every migrated validation key MUST have been observed by the recorder.
	for _, key := range []string{
		"types.validation.description_required",
		"types.validation.id_required",
		"types.validation.name_required",
		"types.validation.model_required",
	} {
		assert.Positivef(t, rec.calls[key], "validation key %q never routed through Tr()", key)
	}
}

// TestRound390_MutationGuard_HardcodedLiteralWouldFail documents the
// paired-mutation contract: if any Validate() reverted to a hardcoded
// English literal, swapping in a non-English Translator would still
// produce English output and this assertion would catch it.
func TestRound390_MutationGuard_HardcodedLiteralWouldFail(t *testing.T) {
	SetTranslator(round390SrTranslator{})
	defer SetTranslator(nil)

	cp := &ConversationPattern{}
	err := cp.Validate()
	require.Error(t, err)
	// A surviving hardcoded "description is required" English literal
	// would make this assertion fail — proving the migration is real.
	assert.True(t, strings.HasPrefix(err.Error(), "sr:"),
		"validation message did not route through Translator (hardcoded literal survived?)")
}

// round390SrTranslator simulates a Serbian locale backend.
type round390SrTranslator struct{}

func (round390SrTranslator) Translate(key string, _ map[string]interface{}) string {
	return "sr:" + key
}

// TestRound390_SetTranslatorNilRestoresDefault asserts nil restores Noop.
func TestRound390_SetTranslatorNilRestoresDefault(t *testing.T) {
	SetTranslator(round390SrTranslator{})
	SetTranslator(nil)
	assert.Equal(t, "description is required", Tr("types.validation.description_required", nil))
}

// TestRound390_ExpandSubstitution covers the {name}-style interpolation
// helper so future keys with placeholders are exercised.
func TestRound390_ExpandSubstitution(t *testing.T) {
	assert.Equal(t, "max 5", expand("max {n}", map[string]interface{}{"n": 5}))
	assert.Equal(t, "no args", expand("no args", nil))
	assert.Equal(t, "id abc", expand("id {x}", map[string]interface{}{"x": "abc"}))
}
