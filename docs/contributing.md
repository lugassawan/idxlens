# Contributing

## Prerequisites

- [Go](https://go.dev/) 1.26+
- [mise](https://mise.jdx.dev/) for tool version management
- [golangci-lint](https://golangci-lint.run/) v2 (installed via mise)

## Development setup

```sh
git clone https://github.com/lugassawan/idxlens.git
cd idxlens
make init
```

`make init` does the following:

1. Trusts and installs mise tool versions (Go, golangci-lint, golines)
2. Builds the custom linter binary (`custom-gcl`)
3. Configures git to use `.githooks/` for hooks

## Available commands

| Command          | Description                                     |
|------------------|-------------------------------------------------|
| `make build`     | Build binary to `bin/idxlens`                   |
| `make lint`      | Build custom linter and run golangci-lint        |
| `make fmt`       | Auto-format code (gofmt + golines)              |
| `make test`      | Run all tests                                   |
| `make coverage`  | Generate coverage report to `coverage/`         |
| `make clean`     | Remove build artifacts                          |

## Code style

- Standard Go conventions: `gofmt` formatting, tab indentation, MixedCaps naming
- Maximum line length: 120 characters (enforced by golines)
- All packages live under `internal/` -- no public API surface
- Error wrapping: `fmt.Errorf("context: %w", err)`

## Testing

- Use the standard library `testing` package only (no testify)
- Write table-driven tests with `t.Run()` subtests
- Place test files alongside source files (`*_test.go`)
- Run tests: `make test` or `go test ./...`
- Generate coverage: `make coverage`

Example test structure:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name string
        input string
        want  string
    }{
        {name: "basic case", input: "a", want: "b"},
        {name: "edge case", input: "", want: ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := doSomething(tt.input)
            if got != tt.want {
                t.Errorf("doSomething(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

## Custom linter

IDXLens uses a custom golangci-lint v2 plugin called [tidygo](https://github.com/lugassawan/tidygo) with five analyzers:

| Analyzer         | Rule                                                     |
|-----------------|----------------------------------------------------------|
| `funcname`       | No underscores in function names                         |
| `maxparams`      | No more than 7 function parameters                       |
| `nolateconst`    | Package-level const/var must appear before functions     |
| `nolocalstruct`  | No named struct declarations inside function bodies      |
| `nolateexport`   | Exported standalone functions must appear before unexported ones |

Always fix lint issues at the root cause. Only use `//nolint` as a last resort with a justification comment.

## Branching and commits

### Branch naming

Use prefixed branch names:

- `feat/` -- new features
- `fix/` -- bug fixes
- `chore/` -- maintenance, tooling, documentation
- `docs/` -- documentation changes

### Commit messages

Follow the `type: description` format. Valid types:

| Type       | Use for                              |
|-----------|--------------------------------------|
| `feat`     | New features                         |
| `fix`      | Bug fixes                            |
| `docs`     | Documentation changes                |
| `refactor` | Code restructuring (no behavior change) |
| `test`     | Adding or updating tests             |
| `build`    | Build system and dependencies        |
| `ci`       | CI/CD configuration                  |
| `chore`    | Maintenance and tooling              |
| `revert`   | Reverting a previous commit          |

Rules:

- No scopes: `feat: add classifier` (not `feat(domain): add classifier`)
- No breaking change marker: `feat: new api` (not `feat!: new api`)
- Enforced by `.githooks/commit-msg`

### Commit splitting

Split changes into logical commits. Separate:

- Infrastructure/config changes
- Core logic
- Tests
- Wiring/integration

Never bundle unrelated changes into a single commit.

## Pull requests

### Direct commits to main/master are blocked

The pre-commit hook prevents direct commits to the main branch. Always work on a feature branch.

### PR title

Use the same `type: description` format as commit messages.

### PR body

Follow the template in `.github/pull_request_template.md`:

```markdown
## Issue
Closes #123

## Summary
- What changed and why

## Test Plan
- [ ] Linter passes (`make lint`)
- [ ] Tests pass (`make test`)

## Notes
Optional context for reviewers.
```

### Before submitting

1. Run the full verification suite:
   ```sh
   make fmt
   make lint
   make test
   ```
2. Ensure all checks pass
3. Push your branch and open a PR

## Architecture

See [architecture.md](architecture.md) for the layer structure. Key rules:

- Dependencies flow downward only: `cli -> output -> domain -> table -> layout -> pdf`
- Cross-layer boundaries use interfaces
- All logic lives in `internal/`
