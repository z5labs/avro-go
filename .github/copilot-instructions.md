# Copilot Instructions

## Project Overview

avro-go is a Go library for working with Avro encoded data. Currently focused on the IDL (Interface Definition Language) parser for Avro schemas.

## Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./idl/...

# Run a single test
go test ./idl -run TestTokenizer

# Lint (CI uses golangci-lint)
golangci-lint run

# Build
go build ./...
```

## Architecture

### IDL Package Pipeline

**Tokenizer → Parser → AST**

1. **Tokenizer** (`tokenizer.go`): Converts IDL source into tokens using `iter.Seq2[Token, error]`
2. **Parser** (`parser.go`): Pulls tokens via `iter.Pull2()` and builds a `File` AST
3. **Printer** (`printer.go`): Formats AST back to IDL text

### State Machine Pattern

Both tokenizer and parser use action functions that return the next action:

```go
type tokenizerAction func(t *tokenizer, yield func(Token, error) bool) tokenizerAction
type parserAction[T any] func(p *parser, t T) (parserAction[T], error)
```

- Return `nil` to end processing
- Capture state via closures when needed

### Type System

Schema types implement the `Type` interface with marker method `idl()`:
`Record`, `Enum`, `Array`, `Map`, `Union`, `Fixed`

### Detailed IDL Guidance

For in-depth patterns when working on the tokenizer or parser (helper functions like `yieldErrorOr`, `yieldTokenThen`, `skipWhitespace`, closure patterns, error types), see `idl/CLAUDE.md`.

## Conventions

### Commit Messages

Format: `{type}({scope}): {description}`
- Types: `feat`, `test`, `refactor`, `fix`
- Scope: Issue number (e.g., `issue-13`)
- Example: `feat(issue-13): implement primitive schema parsing`

### Branch Naming

Feature branches: `story/issue-XX/description`

### Testing

- Table-driven tests with `testCases` slice
- `t.Parallel()` at both test function and subtest level
- Use `github.com/stretchr/testify/require` (not `assert`)
- Test case names are descriptive and lowercase

```go
func TestExample(t *testing.T) {
    t.Parallel()

    testCases := []struct {
        name     string
        input    string
        expected []Thing
    }{
        {
            name:     "descriptive lowercase name",
            input:    `example input`,
            expected: []Thing{...},
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            // ...
            require.NoError(t, err)
            require.Equal(t, tc.expected, result)
        })
    }
}
```
