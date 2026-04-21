# I-LLM

Interactive LLM conversation patterns and structured-reasoning
frameworks: chain-of-thought, tree-of-thought, ReAct, few-shot, and a
step-by-step prompt chain runner. Part of the Plinius Go service family
used by HelixAgent.

## Status

- Compiles: `go build ./...` exits 0.
- Tests pass under `-race`: 2 packages (types, client), all green.
- Default library seeded on `New()`: 3 patterns (cot-basic, react-basic,
  few-shot) and 1 chain (summarise-then-translate).
- Integration-ready: consumable Go library for the HelixAgent ensemble.

## Purpose

- `pkg/types` — value types: `ConversationPattern`, `ReActStep`,
  `AgentConfig`, `Tool`, `ChainResult`, `PromptChain`, `ChainStep`,
  `Agent`, `TreeResult`.
- `pkg/client` — pattern / chain orchestration:
  - `GetPattern`, `ListPatterns`, `RenderPattern`
  - `CreateAgent(AgentConfig)`
  - `RunChain(chain, inputs)` — executes a multi-step chain with
    variable propagation
  - `ChainOfThought(problem, model)`
  - `TreeOfThought(problem, model, breadth)` — parallel branching
  - `GetCategories`
  - `RegisterPattern`, `RegisterChain`, `RegisterTool`, `SetRunner`

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

c.SetRunner(func(ctx context.Context, prompt string) (string, error) {
    // call into an LLM provider of your choice
    return "...", nil
})

r, err := c.ChainOfThought(context.Background(), "What is 17 * 23?", "gpt-4")
if err != nil { log.Fatal(err) }
log.Println(r.FinalOutput)
```

## Module path

```go
import "digital.vasic.illm"
```

## Lineage

Extracted from internal HelixAgent research tree on 2026-04-21.
Graduated to functional status on the next day alongside its 7 sibling
Plinius modules.

Historical research corpus (unused) remains at
`docs/research/go-elder-plinius-v3/go-elder-plinius/go-i-llm/` inside
the HelixAgent repository.

## Development layout

This module's `go.mod` declares the module as `digital.vasic.illm` and
uses a relative `replace` directive pointing at `../PliniusCommon`.

## License

Apache-2.0
