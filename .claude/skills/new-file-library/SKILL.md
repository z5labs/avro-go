---
name: new-file-library
description: Scaffold a new file library package with tokenizer, parser, printer, and tests
disable-model-invocation: true
argument-hint: "[package-name]"
---

Scaffold a new file library package at `./$ARGUMENTS[0]/` following the established pipeline pattern from the `idl/` package. The package name is `$ARGUMENTS[0]`.

## What to Generate

Create the following files. Read the corresponding `idl/` files first to match the exact patterns, then adapt them for the new package.

### 1. `$ARGUMENTS[0]/doc.go`

```go
// Copyright (c) !`date +%Y` Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

// Package $ARGUMENTS[0] provides tools for working with [FORMAT DESCRIPTION].
package $ARGUMENTS[0]
```

### 2. `$ARGUMENTS[0]/tokenizer.go`

Scaffold with:
- `Pos` struct (Line, Column int)
- `Token` struct (Pos, Type, Value) with `String()` method
- `TokenType` enum with at minimum: `TokenComment`, `TokenIdentifier`, `TokenSymbol`, `TokenString`, `TokenNumber`
- `tokenizer` struct wrapping a `*bufio.Reader` with position tracking and `next()`, `backup()` methods
- `tokenizerAction` type: `func(t *tokenizer, yield func(Token, error) bool) tokenizerAction`
- Helper functions: `yieldErrorOr`, `yieldTokenThen`, `skipWhitespace`
- `Tokenize` public function returning `iter.Seq2[Token, error]`
- Stub `tokenize$ARGUMENTS[0]` entry point action (capitalize first letter) that reads one rune and returns nil
- Error types: `UnexpectedCharacterError`

### 3. `$ARGUMENTS[0]/tokenizer_test.go`

Scaffold with:
- `collect` helper function for gathering tokens from `iter.Seq2`
- One example table-driven test (`TestTokenizer`) with a single placeholder test case
- `t.Parallel()` at both levels
- `require` from testify

### 4. `$ARGUMENTS[0]/parser.go`

Scaffold with:
- `File` struct as the top-level AST node (with at minimum a placeholder field)
- `Type` interface with marker method
- `parser` struct wrapping `next func() (Token, error, bool)` with `expect()` method
- `parserAction[T]` type: `func(p *parser, t T) (parserAction[T], error)`
- `Parse` public function: creates parser via `iter.Pull2(Tokenize(r))`, runs action loop, returns `*File`
- Stub `parseFile` entry action that returns `(nil, nil)`
- Error types: `UnexpectedEndOfTokensError`, `UnexpectedTokenError`

### 5. `$ARGUMENTS[0]/parser_test.go`

Scaffold with:
- One example table-driven test (`TestParser`) with a single placeholder test case
- Tests MUST call `Parse()` with real source strings, not construct AST manually
- `t.Parallel()` at both levels
- `require` from testify

### 6. `$ARGUMENTS[0]/printer.go`

Scaffold with:
- `printer` struct wrapping `io.Writer` with `err` field, `write()`, `writef()` methods
- `printerAction` type: `func(pr *printer, f *File) printerAction`
- `writeThen` helper function
- `Print` public function: runs action loop checking `pr.err` each iteration
- Stub `printFile` entry action that returns nil

### 7. `$ARGUMENTS[0]/printer_test.go`

Scaffold with:
- One example table-driven test (`TestPrinter`) with direct print test structure
- One example round-trip test (`TestPrinterRoundTrip`) structure
- `t.Parallel()` at both levels
- `require` from testify

### 8. `$ARGUMENTS[0]/CLAUDE.md`

Create a package-specific CLAUDE.md following the structure of `idl/CLAUDE.md`. Include:
- State machine pattern documentation with the package's action types
- Helper function documentation
- Testing style documentation
- Error types

## After Scaffolding

1. Run `go mod tidy` to update dependencies
2. Run `go build ./$ARGUMENTS[0]/...` to verify compilation
3. Run `go test ./$ARGUMENTS[0]/...` to verify tests pass
4. Report what was created and what the user should implement next
