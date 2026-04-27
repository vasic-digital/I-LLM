# CLAUDE.md -- digital.vasic.illm


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Patterns (few-shot / CoT / ToT / ReAct) + chain execution with injected Runner
cd I-LLM && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./pkg/client
```
Expect: PASS; default patterns seeded, `RunChain` threads variables through steps, `CreateAgent` wires tools.


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

## API Cheat Sheet

**Module path:** `digital.vasic.illm`.

```go
type Runner      func(ctx, prompt string) (string, error)
type ToolHandler func(ctx, name, input string) (string, error)

type ConversationPattern struct {
    ID, Name, Description, Category, Template string
    Variables []string
}
type PromptChain struct {
    ID, Name, Description string
    Steps []ChainStep
}
type ChainStep struct {
    Name, Template, OutputKey string
    Variables []string
}
type AgentConfig struct {
    Name, Objective string
    MaxSteps int
    Tools []string
}
type ChainResult struct {
    Success bool
    Output  string
    Variables map[string]string
}

type Client struct { /* patterns, chains, tools, runner */ }

func New(opts ...config.Option) (*Client, error)
func (c *Client) SetRunner(r Runner)
func (c *Client) RegisterPattern(p ConversationPattern) error
func (c *Client) RegisterChain(p PromptChain) error
func (c *Client) RegisterTool(name string, h ToolHandler)
func (c *Client) RenderPattern(ctx, pattern ConversationPattern, vars map[string]string) (string, error)
func (c *Client) CreateAgent(ctx, cfg AgentConfig) (*Agent, error)
func (c *Client) RunChain(ctx, chain PromptChain, inputs map[string]string) (*ChainResult, error)
func (c *Client) ChainOfThought(ctx, prompt string) (*ChainResult, error)
func (c *Client) Close() error
```

**Typical usage:**
```go
c, _ := illm.New()
defer c.Close()
c.SetRunner(func(ctx context.Context, prompt string) (string, error) { return llm.Complete(prompt) })
result, _ := c.RunChain(ctx, myChain, map[string]string{"input": "analyze X"})
```

**Injection points:** `Runner` (LLM completion), `ToolHandler` (ReAct tool calls).
**Defaults on `New`:** few-shot, CoT, ToT, ReAct patterns + 1 demo chain.

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | PliniusCommon |
| Downstream (these import this module) | root only |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->

