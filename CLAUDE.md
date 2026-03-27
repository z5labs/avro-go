# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

avro-go is a Go library for working with Avro encoded data. Currently focused on building an IDL (Interface Definition Language) parser for Avro schemas.

## Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./idl/...

# Run a single test
go test ./idl -run TestTokenizer

# Run tests with verbose output
go test ./... -v

# Format code
go fmt ./...

# Build
go build ./...
```

## Architecture

### `idl` Package

The IDL package provides tools for parsing Avro IDL files. It follows a pipeline architecture:

**Tokenizer → Parser → AST**

1. **Tokenizer** (`tokenizer.go`): Converts IDL source into tokens using `iter.Seq2[Token, error]`. Token types include comments, identifiers, symbols, strings, and numbers.

2. **Parser** (`parser.go`): Converts tokens into an AST. Uses `iter.Pull2()` to pull tokens on demand. Outputs a `File` containing schemas and protocols.

3. **Printer** (`printer.go`): Formats AST back to IDL text (currently a stub).

### State Machine Pattern

Both tokenizer and parser use a recursive action function pattern:
```go
type tokenizerAction func(t *tokenizer, yield func(Token, error) bool) tokenizerAction
type parserAction[T any] func(p *parser, t T) (parserAction[T], error)
```

Functions return the next action to execute, enabling clean sequential processing.

### Type System

Schema types implement the `Type` interface with a marker method `idl()`:
- `Record`, `Enum`, `Array`, `Map`, `Union`, `Fixed`

## Commit Convention

Format: `{type}({scope}): {description}`
- Types: `feat`, `test`, `refactor`, `fix`
- Scope: Issue number (e.g., `issue-13`)
- Example: `feat(issue-13): implement primitive schema parsing`

## Branch Naming

Feature branches: `story/issue-XX/description`

## Testing Style

Tests use table-driven patterns with parallel execution:

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

            result, err := DoThing(tc.input)

            require.NoError(t, err)
            require.Equal(t, tc.expected, result)
        })
    }
}
```

Key patterns:
- `t.Parallel()` at both test function and subtest level
- Table-driven tests with `testCases` slice containing `name`, inputs, and `expected` output
- Subtests via `t.Run(tc.name, ...)`
- Assertions with `github.com/stretchr/testify/require` (not `assert`)
- Test case names are descriptive and lowercase
