// Package types — CONST-046 i18n seam.
//
// This file establishes the hardcoded-content abstraction for the I-LLM
// library (round-390 §11.4 anti-bluff sweep, 2026-05-20). Every
// user-facing label, description, and validation message the library
// emits is routed through tr() so a consuming application can render it
// in the end user's language instead of the built-in English fallback.
//
// The seam lives in the lowest-level package (types) so both
// pkg/types and pkg/client funnel their user-facing strings through a
// single Translator. It mirrors the proven self-contained pattern of
// the sibling vasic-digital/Benchmark module's benchmark/i18n.go:
// a NoopTranslator with a built-in English bundle, swappable via
// SetTranslator.
//
// CONST-051(B) decoupling: the I-LLM module stays project-not-aware. It
// ships only NoopTranslator (English fallback, key-stable). A consuming
// project injects its own i18n backend via SetTranslator — the module
// never reaches into a parent's resource tree.
package types

import "sync"

// Translator resolves a message key (and optional named arguments) into
// a locale-appropriate string. It is the CONST-046 seam for
// digital.vasic.illm.
type Translator interface {
	// Translate returns the localized string for key. args holds named
	// substitution values (e.g. {"max": 3}); a backend that does not
	// support substitution may ignore them. An unknown key MUST fall
	// back to a stable, non-empty string (NoopTranslator returns the
	// built-in English default, or the key itself when unknown).
	Translate(key string, args map[string]interface{}) string
}

// NoopTranslator is the default Translator. It returns the built-in
// English default for every known key and the key itself for any
// unknown key, so the library is fully usable with no i18n backend
// wired. It performs no locale negotiation — that is the consuming
// project's responsibility.
type NoopTranslator struct{}

// Translate implements Translator using the built-in English bundle.
func (NoopTranslator) Translate(key string, args map[string]interface{}) string {
	if msg, ok := enBundle[key]; ok {
		return expand(msg, args)
	}
	return key
}

var (
	translatorMu     sync.RWMutex
	activeTranslator Translator = NoopTranslator{}
)

// SetTranslator installs a Translator for the process. Passing nil
// restores the built-in NoopTranslator. Safe for concurrent use; a
// consuming application typically calls this once at startup after
// negotiating the user's locale.
func SetTranslator(t Translator) {
	translatorMu.Lock()
	defer translatorMu.Unlock()
	if t == nil {
		activeTranslator = NoopTranslator{}
		return
	}
	activeTranslator = t
}

// Tr resolves key through the active Translator. It is the single
// exported entry point every user-facing string in the I-LLM library
// funnels through (pkg/client imports it from here).
func Tr(key string, args map[string]interface{}) string {
	translatorMu.RLock()
	t := activeTranslator
	translatorMu.RUnlock()
	return t.Translate(key, args)
}

// expand performs minimal {name}-style substitution on a bundle
// template so the built-in English fallback can render counts and
// identifiers without pulling a templating dependency into the
// decoupled module.
func expand(msg string, args map[string]interface{}) string {
	if len(args) == 0 {
		return msg
	}
	out := msg
	for k, v := range args {
		out = replaceAll(out, "{"+k+"}", toStr(v))
	}
	return out
}

func replaceAll(s, old, repl string) string {
	if old == "" {
		return s
	}
	var b []byte
	for {
		i := indexOf(s, old)
		if i < 0 {
			b = append(b, s...)
			break
		}
		b = append(b, s[:i]...)
		b = append(b, repl...)
		s = s[i+len(old):]
	}
	return string(b)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toStr(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return itoa(x)
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// enBundle is the built-in English message bundle. A consuming project's
// Translator overrides any subset of these keys for its locales; keys
// absent from a custom backend fall back here. Keys are stable
// identifiers — never change a key without updating every tr() call
// site, the paired-mutation tests, and consuming overrides.
var enBundle = map[string]string{
	// --- validation messages (pkg/types) ---
	"types.validation.description_required": "description is required",
	"types.validation.id_required":          "id is required",
	"types.validation.name_required":        "name is required",
	"types.validation.model_required":       "model is required",

	// --- client error messages (pkg/client) ---
	"client.err.invalid_configuration": "invalid configuration",
	"client.err.invalid_pattern":       "invalid pattern",
	"client.err.invalid_chain":         "invalid chain",
	"client.err.invalid_agent_config":  "invalid agent config",
	"client.err.pattern_not_found":     "pattern not found",
	"client.err.problem_required":      "problem is required",
	"client.err.runner_failed":         "runner failed",

	// --- built-in pattern catalogue (pkg/client seedDefaults) ---
	"client.pattern.cot.name":     "Chain-of-Thought",
	"client.pattern.cot.desc":     "Basic step-by-step reasoning scaffold.",
	"client.pattern.react.name":   "ReAct",
	"client.pattern.react.desc":   "ReAct agent scaffold with Thought/Action/Observation.",
	"client.pattern.fewshot.name": "Few-Shot",
	"client.pattern.fewshot.desc": "Few-shot example scaffold.",
	"client.chain.summtrans.name": "Summarise then Translate",
	"client.chain.summtrans.desc": "Two-step chain: summarise input, then translate the summary.",
	"client.step.cot_thought":     "Chain-of-thought decomposition",
}
