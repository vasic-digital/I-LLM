# I-LLM

Interactive LLM conversation patterns and structured-reasoning
frameworks: chain-of-thought (CoT), tree-of-thought (ToT), ReAct,
few-shot, and a step-by-step prompt-chain runner. Part of the
Plinius Go service family used by the HelixAgent ensemble.

Module path: `digital.vasic.illm` — two packages: `pkg/types` (value
types) and `pkg/client` (orchestration).

## Status

- **FUNCTIONAL** — both packages ship tested implementations.
- `go test -race -count=1 ./...` is green (round-265 evidence in
  `docs/test-coverage.md`).
- Default library seeded on `New()`: 3 patterns (`cot-basic`,
  `react-basic`, `few-shot`) and 1 chain
  (`summarise-then-translate`).
- Default `Runner` returns `ErrBaselineRunnerNotConfigured` —
  callers MUST inject a real LLM-dispatching `Runner` via
  `SetRunner` before invoking `ChainOfThought` / `TreeOfThought` /
  `RunChain` (round-23 §11.4 audit fix; do not regress).
- Integration-ready: consumable Go library for HelixAgent.

## Public surface

`pkg/types` — value types: `ConversationPattern`, `ReActStep`,
`AgentConfig`, `Tool`, `ChainResult`, `PromptChain`, `ChainStep`,
`Agent`, `TreeResult` (each carrying a `Validate()` and, for
`AgentConfig`, a `Defaults()`).

`pkg/client` — orchestration:

- `New(opts ...config.Option) (*Client, error)` /
  `NewFromConfig(cfg *config.Config) (*Client, error)`
- `(*Client).Close() error` / `(*Client).Config() *config.Config`
- `(*Client).SetRunner(r Runner)` — inject the real LLM-dispatching
  function used by CoT / ToT / RunChain
- `(*Client).RegisterPattern(p ConversationPattern) error`
- `(*Client).RegisterChain(p PromptChain) error`
- `(*Client).RegisterTool(name string, h ToolHandler)`
- `(*Client).GetPattern(ctx, id) (*ConversationPattern, error)`
- `(*Client).ListPatterns(ctx, category) ([]ConversationPattern, error)`
- `(*Client).RenderPattern(ctx, pattern, vars) (string, error)`
- `(*Client).CreateAgent(ctx, AgentConfig) (*Agent, error)`
- `(*Client).RunChain(ctx, chain, inputs) (*ChainResult, error)`
- `(*Client).ChainOfThought(ctx, problem, model) (*ChainResult, error)`
- `(*Client).TreeOfThought(ctx, problem, model, breadth) (*TreeResult, error)`
- `(*Client).GetCategories(ctx) ([]string, error)`
- Sentinel `ErrBaselineRunnerNotConfigured`

## Usage

```go
import (
    "context"
    "log"

    illm "digital.vasic.illm/pkg/client"
    "digital.vasic.illm/pkg/types"
)

c, err := illm.New()
if err != nil { log.Fatal(err) }
defer c.Close()

// REQUIRED: inject a real LLM Runner. Without this,
// CoT / ToT / RunChain return ErrBaselineRunnerNotConfigured
// (round-23 §11.4 audit fix — no fabricated reasoning by default).
c.SetRunner(func(ctx context.Context, prompt string) (string, error) {
    return provider.Complete(ctx, prompt)
})

r, err := c.ChainOfThought(context.Background(), "What is 17 * 23?", "gpt-4")
if err != nil { log.Fatal(err) }
log.Println(r.FinalOutput)

// Multi-step chain with variable propagation
chain := types.PromptChain{
    ID: "my-chain", Name: "demo", Description: "two-step",
    Steps: []types.ChainStep{
        {Name: "summarise", PromptTemplate: "Summarise: {{text}}", OutputKey: "summary"},
        {Name: "translate", PromptTemplate: "Translate to {{lang}}: {{summary}}", OutputKey: "translated"},
    },
}
res, _ := c.RunChain(ctx, chain, map[string]string{"text": "...", "lang": "ja"})
log.Println(res.FinalOutput)
```

## Anti-bluff guarantees (round-265)

Every PASS produced by this submodule's tests + Challenges carries
positive runtime evidence per Article XI §11.9 and the verbatim
2026-05-19 operator mandate:

> "all existing tests and Challenges do work in anti-bluff manner —
> they MUST confirm that all tested codebase really works as
> expected! We had been in position that all tests do execute with
> success and all Challenges as well, but in reality the most of
> the features does not work and can't be used! This MUST NOT be
> the case and execution of tests and Challenges MUST guarantee
> the quality, the completition and full usability by end users
> of the product!"

Seven invariants enforced by the round-265 runner +
`i_llm_describe_challenge.sh` paired-mutation gate:

1. **Default-surface coverage.** `client.New` must seed 3
   patterns (`cot-basic`, `react-basic`, `few-shot`) and 1 chain
   (`summarise-then-translate`). Missing-id lookups MUST surface
   `ErrCodeNotFound`, not silently return nil.
2. **Byte-exact substitution.** `RenderPattern` must round-trip
   the substituted variables byte-exact across 5 locales (en, sr
   Cyrillic, ja, ar RTL, zh-CN Han). Runner asserts via
   `strings.Contains` plus rune-count comparison to the fixture's
   `expected_min_runes`.
3. **Dispatch byte-equality.** `ChainOfThought` must dispatch the
   prompt to the injected `Runner` byte-exact — the runner uses a
   capturing `Runner` that records the most-recent prompt and
   asserts equality against the fixture-derived problem statement
   across 5 locales.
4. **Tree branching invariant.** `TreeOfThought(problem, model, 3)`
   must produce `Breadth=3`, `len(Branches)=3`, and an aggregated
   `FinalOutput` that contains the locale's problem string. Every
   branch's `FinalOutput` is independently asserted to contain the
   locale problem before the aggregation check.
5. **Chain variable propagation.** A 2-step `RunChain` MUST write
   step 1's output under its `OutputKey` and reference it
   literally in step 2's prompt template. Runner asserts the final
   output contains the doubly-nested `STEP_OUT[Summarise: …]`
   marker plus the locale-specific `lang` variable.
6. **Sentinel-default re-validation.** A fresh `client.New` WITHOUT
   `SetRunner` MUST surface `ErrBaselineRunnerNotConfigured`
   (wrapped as `ErrCodeUnavailable` from `RunChain`) — round-23
   §11.4 audit fix re-validated for all three dispatch paths
   (CoT, ToT, RunChain).
7. **Paired mutation.** Running the describe gate with
   `--anti-bluff-mutate` plants a deliberate symbol-rename in a
   tmp copy of `docs/test-coverage.md`
   (`ChainOfThought -> ChainOfThought_MUTATED`), reruns the
   structural cross-reference check, and asserts the gate exits
   99. Proves the ledger-to-source map actually catches drift
   instead of rubber-stamping it.

A Section that returns success without producing the corresponding
PASS line is a §11.9 violation regardless of how green the summary
line looks.

## Test bank

```bash
# Unit tests (CONST-050(A) — mocks allowed only here)
GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./...

# Round-265 challenge runner (real client, capturing Runner, 5 locales)
go run ./challenges/runner/ -fixtures tests/fixtures/i_llm/payloads.json

# Describe challenge — clean mode (exit 0)
bash challenges/scripts/i_llm_describe_challenge.sh

# Paired-mutation gate (must exit 99)
bash challenges/scripts/i_llm_describe_challenge.sh --anti-bluff-mutate

# Inherited governance challenges
bash challenges/scripts/no_suspend_calls_challenge.sh
bash challenges/scripts/host_no_auto_suspend_challenge.sh
bash challenges/scripts/chaos_failure_injection_challenge.sh
bash challenges/scripts/ddos_health_flood_challenge.sh
bash challenges/scripts/scaling_horizontal_challenge.sh
bash challenges/scripts/stress_sustained_load_challenge.sh
bash challenges/scripts/ui_terminal_interaction_challenge.sh
bash challenges/scripts/ux_end_to_end_flow_challenge.sh
```

The round-265 runner exits non-zero on any FAIL; the symbol-to-test
ledger lives in `docs/test-coverage.md`.

## Module path & development layout

```go
import "digital.vasic.illm"
```

`go.mod` declares the module as `digital.vasic.illm` and uses a
relative `replace` directive pointing at `../PliniusCommon`. The
challenge runner `challenges/runner/main.go` lives under the same
module — `go build ./challenges/runner/` from the repo root is
sufficient to produce the runner binary at `/tmp/`.

## Lineage

Extracted from internal HelixAgent research tree on 2026-04-21,
graduated to functional status the next day alongside its 7 sibling
Plinius modules. Round-23 §11.4 audit (2026-05-17) removed the
silent "RSP:"-prefix baseline `Runner` after it was found to be
producing fabricated reasoning when downstream consumers forgot to
call `SetRunner`. Round-265 (2026-05-19) adds the deep-doc ledger,
the multi-locale challenge runner, and the paired-mutation describe
gate.

Historical research corpus (unused) remains at
`docs/research/go-elder-plinius-v3/go-elder-plinius/go-i-llm/` inside
the HelixAgent repository.

## Governance

This submodule inherits the constitution submodule's universal
rules. See `CLAUDE.md`, `AGENTS.md`, `CONSTITUTION.md` for the
cascaded clauses (CONST-033, CONST-035, CONST-036, CONST-042,
CONST-043, CONST-047..061).

## License

Apache-2.0
