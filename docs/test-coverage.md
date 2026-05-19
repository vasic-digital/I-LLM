# Test-Coverage Ledger — round-265

This ledger maps every exported symbol of `digital.vasic.illm`
to the test or Challenge that exercises it with captured runtime
evidence. Per CONST-035, CONST-050(B), and the 2026-05-19 operator
mandate quoted below, no symbol may PASS without a corresponding
runtime-evidence exercise.

> Verbatim 2026-05-19 operator mandate: "all existing tests and
> Challenges do work in anti-bluff manner - they MUST confirm that
> all tested codebase really works as expected! We had been in
> position that all tests do execute with success and all
> Challenges as well, but in reality the most of the features does
> not work and can't be used! This MUST NOT be the case and
> execution of tests and Challenges MUST guarantee the quality, the
> completition and full usability by end users of the product!"

Operative rule (Article XI §11.9): **The bar for shipping is not
"tests pass" but "users can use the feature."** Every PASS in the
table below carries either a unit test, an integration test, or a
challenge-runner section that produces positive runtime evidence —
no metadata-only / grep-only PASS counts.

## Module surface

`digital.vasic.illm` ships two Go packages:

- **`pkg/types`** — value types: `ConversationPattern`, `ReActStep`,
  `AgentConfig`, `Tool`, `ChainResult`, `PromptChain`, `ChainStep`,
  `Agent`, `TreeResult`. All have `Validate()` invariants where
  applicable; `AgentConfig.Defaults()` installs Temperature=0.7
  when zero.
- **`pkg/client`** — pattern/chain orchestration: `Client`,
  `New`, `NewFromConfig`, `Close`, `Config`, `SetRunner`,
  `RegisterPattern`, `RegisterChain`, `RegisterTool`, `GetPattern`,
  `ListPatterns`, `RenderPattern`, `CreateAgent`, `RunChain`,
  `ChainOfThought`, `TreeOfThought`, `GetCategories`. Two function
  types — `Runner`, `ToolHandler`. One sentinel —
  `ErrBaselineRunnerNotConfigured`.

## Symbol → exerciser map

### `pkg/types` (`types.go`)

| Symbol | Kind | Exercised by |
|--------|------|--------------|
| `ConversationPattern` | struct | runner Section 1 (3 seeded patterns retrieved) + Section 2 (RenderPattern uses `few-shot`) + `pkg/types/types_test.go` |
| `ConversationPattern.Validate` | method | `pkg/types/types_test.go` (empty Description / ID / Name rejected) |
| `ReActStep` | struct | runner Section 3 (`ChainOfThought` produces a 1-step Steps slice; ActionInput asserted equal to fixture-derived problem) |
| `AgentConfig` | struct | runner Section 6 (CreateAgent + invalid AgentConfig rejection) + `pkg/types/types_test.go` |
| `AgentConfig.Validate` | method | runner Section 6 (Model="" rejected) + `pkg/types/types_test.go` |
| `AgentConfig.Defaults` | method | runner Section 6 (Temperature 0 -> 0.7 verified post-CreateAgent) + `pkg/types/types_test.go` |
| `Tool` | struct | `pkg/types/types_test.go` (validation rules) |
| `Tool.Validate` | method | `pkg/types/types_test.go` |
| `ChainResult` | struct | runner Section 3 (CoT returns `ChainResult` with Iterations=1, Steps[0]) + Section 5 (RunChain returns Iterations=2) |
| `PromptChain` | struct | runner Section 5 (custom `round265-locale-chain` registered + executed) + default seed `summarise-then-translate` |
| `PromptChain.Validate` | method | `pkg/client/client_test.go` (invalid chain rejected by RegisterChain) |
| `Agent` | struct | runner Section 6 (Agent.ID prefix `agent-`, Agent.Config.Temperature checked) |
| `TreeResult` | struct | runner Section 4 (Breadth=3, len(Branches)=3, FinalOutput aggregated from longest branch) |
| `ChainStep` | struct | runner Section 5 (2-step chain — OutputKey propagation from step1 to step2) |
| `ChainStep.Validate` | method | `pkg/client/client_test.go` (invalid step name -> chain rejected) |

### `pkg/client` (`client.go`)

| Symbol | Kind | Exercised by |
|--------|------|--------------|
| `Runner` | type alias | runner Section 3+4+5 (capturingRunner.Run wired via SetRunner) |
| `ToolHandler` | type alias | runner Section 6 (RegisterTool stores a real handler) |
| `Client` | struct | runner Sections 1-7 |
| `New` | func | runner Sections 1-7 (every section constructs a fresh client) + `pkg/client/client_test.go` |
| `NewFromConfig` | func | `pkg/client/client_test.go` (Config injection path) |
| `Client.Close` | method | runner Sections 1-7 (defer Close + double-close idempotency) + `pkg/client/client_test.go` |
| `Client.Config` | method | runner Section 1 (non-nil cfg asserted) |
| `Client.SetRunner` | method | runner Section 3+4+5 (capturingRunner installed; nil-input no-op covered in Section 6 indirectly) |
| `Client.RegisterPattern` | method | `pkg/client/client_test.go` (`TestRegisterPattern`) |
| `Client.RegisterChain` | method | runner Section 5 (custom chain registered) + `pkg/client/client_test.go` |
| `Client.RegisterTool` | method | runner Section 6 (named handler stored + nil-input no-op) |
| `Client.GetPattern` | method | runner Section 1 (3 seeded ids + 1 missing) + `pkg/client/client_test.go` |
| `Client.ListPatterns` | method | runner Section 1 (all + filtered by category="reasoning") + `pkg/client/client_test.go` |
| `Client.RenderPattern` | method | runner Section 2 (5 locales, byte-exact substitution) + `pkg/client/client_test.go` |
| `Client.CreateAgent` | method | runner Section 6 (valid + invalid + Defaults applied) + `pkg/client/client_test.go` |
| `Client.RunChain` | method | runner Section 5 (2-step propagation) + `pkg/client/client_test.go` + `client_extra_test.go` |
| `Client.ChainOfThought` | method | runner Section 3 (5 locales, capturing Runner round-trip) + `pkg/client/client_test.go` |
| `Client.TreeOfThought` | method | runner Section 4 (5 locales × 3 branches) + `pkg/client/client_test.go` |
| `Client.GetCategories` | method | runner Section 1 + `pkg/client/client_test.go` |
| `ErrBaselineRunnerNotConfigured` | var | runner Section 7 (CoT + ToT + RunChain ALL surface sentinel without SetRunner) + `pkg/client/client_test.go` (TestChainOfThoughtWithoutInjectedRunner_ReturnsSentinel + TestTreeOfThoughtWithoutInjectedRunner_ReturnsSentinel + TestRunChainWithoutInjectedRunner_ReturnsSentinel) |

## Test runs (round-265 evidence captured)

### `go test -race -count=1 ./...`

```
ok  	digital.vasic.illm/pkg/client	(race ~1s)
ok  	digital.vasic.illm/pkg/types	(race ~1s)
```

Both packages pass with `-race` enabled — no data-race detected at
the Runner mutex, the patterns/chains/tools maps, or the
`Client.closed` flag.

### `challenges/runner/main.go -fixtures tests/fixtures/i_llm/payloads.json`

```
=== Round-265 I-LLM Challenge Runner ===
... 37 PASS lines across 7 sections, 5 locales ...
=== Summary: 37 PASS, 0 FAIL ===
```

Per-locale runtime evidence captured:

- Section 1: 9 default-surface PASS — 3 seeded patterns + 3
  retrievals + 1 missing-id sentinel + GetCategories + filtered
  ListPatterns.
- Section 2: 5 RenderPattern PASS — byte-exact substitution
  verified per locale (rune counts captured).
- Section 3: 5 ChainOfThought PASS — capturing Runner asserts
  byte-exact prompt dispatch + round-trip in 5 locales.
- Section 4: 5 TreeOfThought PASS — every branch in every locale
  echoes the locale problem; longest-output aggregation honoured.
- Section 5: 1 RegisterChain + 5 RunChain PASS — step1 output
  literally propagates into step2 template (`STEP_OUT[Summarise: …]`).
- Section 6: 4 PASS — CreateAgent + Defaults + invalid rejection
  + RegisterTool (named) + nil-input no-op.
- Section 7: 3 PASS — ErrBaselineRunnerNotConfigured re-validated
  for CoT, ToT, RunChain dispatch paths.

### `bash challenges/scripts/i_llm_describe_challenge.sh`

Clean mode exit 0; `--anti-bluff-mutate` exit 99 (paired mutation
correctly detected — ledger-vs-source drift caught).

## Anti-bluff invariants

This round addresses every taxonomy entry in CLAUDE.md §"Bluff
taxonomy":

- **Wrapper bluff** — the describe-challenge wrapper uses
  PASS/FAIL counters with a separate `set -uo pipefail` guard, never
  inline arithmetic on a command that prints + exits non-zero.
- **Contract bluff** — every public method on `Client` and every
  exported type listed above is exercised by a runtime test or
  challenge section. The ledger surface is closed and audited.
- **Structural bluff** — no `check_file_exists` PASS without a
  paired functional assertion. Every PASS carries either a rune
  count, a token count, a branch count, a round-trip equality, or
  an `errors.Is` sentinel match.
- **Comment bluff** — the README's `## Anti-bluff guarantees`
  section is enforced by `i_llm_describe_challenge.sh` Section 5.
- **Skip bluff** — no `t.Skip()` in the unit tests; the runner has
  no `if false { … }` dead branches.

## Cross-reference to constitutional anchors

| Anchor | Layer | How honoured |
|--------|-------|--------------|
| CONST-035 / Article XI §11.9 | end-user-usability | every PASS line carries runtime evidence (locale, rune count, token count, sentinel match) |
| CONST-050(A) | no-fakes-beyond-unit-tests | runner uses only the public client API; the capturingRunner is the consumer's injected dependency, NOT a library-internal mock |
| CONST-050(B) | 100%-test-type coverage | unit tests + challenge runner + paired-mutation gate together cover unit + integration-style + meta-test layers |
| CONST-053 | .gitignore | `.gitignore` covers `/bin/`, `*.test`, `coverage.out`, IDE state |

The 2026-05-19 operator mandate is preserved verbatim above and in
the runner's package doc comment.
