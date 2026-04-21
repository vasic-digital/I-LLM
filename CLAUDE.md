# CLAUDE.md -- digital.vasic.illm

Module-specific guidance for Claude Code.

## Status

**FUNCTIONAL.** 2 packages (types, client) ship tested implementations;
`go test -race ./...` all green. Default pattern library + 1 demo
chain seeded on `New()`. Chain-of-thought, tree-of-thought, and
step-by-step chain runner implemented against an injectable Runner.

## Hard rules

1. **NO CI/CD pipelines** -- no `.github/workflows/`, `.gitlab-ci.yml`,
   `Jenkinsfile`, `.travis.yml`, `.circleci/`, or any automated
   pipeline. No Git hooks either. Permanent.
2. **SSH-only for Git** -- `git@github.com:...` / `git@gitlab.com:...`.
3. **Conventional Commits** -- `feat(i-llm): ...`, `fix(...)`,
   `docs(...)`, `test(...)`, `refactor(...)`.
4. **Code style** -- `gofmt`, `goimports`, 100-char line ceiling,
   errors always checked and wrapped (`fmt.Errorf("...: %w", err)`).
5. **Resource cap for tests** --
   `GOMAXPROCS=2 nice -n 19 ionice -c 3 go test -count=1 -p 1 -race ./...`

## Purpose

Interactive LLM conversation patterns and structured reasoning
frameworks. Key surface: `GetPattern`, `ListPatterns`, `RenderPattern`,
`CreateAgent`, `RunChain`, `ChainOfThought`, `TreeOfThought`,
`GetCategories`, `RegisterPattern`, `RegisterChain`, `RegisterTool`,
`SetRunner`.

## Primary consumer

HelixAgent (`dev.helix.agent`).

## Testing

```
GOMAXPROCS=2 nice -n 19 ionice -c 3 go test -count=1 -p 1 -race ./...
```
