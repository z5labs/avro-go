# IDL Package - Claude Memory

This file documents the coding style and patterns specific to the IDL tokenizer and parser implementation.

## State Machine Pattern

Both tokenizer and parser use a recursive action function pattern for clean sequential processing.

### Tokenizer Actions

```go
type tokenizerAction func(t *tokenizer, yield func(Token, error) bool) tokenizerAction
```

- Each action function returns the next action to execute
- Return `nil` to end iteration
- The `yield` function follows Go iterator conventions: return `false` to stop early

### Parser Actions

```go
type parserAction[T any] func(p *parser, t T) (parserAction[T], error)
```

- Generic over the type being built (e.g., `*File`, `*Schema`)
- Returns both the next action and any error
- Return `(nil, nil)` to complete successfully
- Return `(nil, err)` to terminate with error

## Tokenizer Helper Functions

### `yieldErrorOr(err error, next tokenizerAction) tokenizerAction`

Handles error propagation in the tokenizer chain:
- If `err == nil`, returns `next` action
- If `err == io.ErrUnexpectedEOF`, returns `nil` (clean termination)
- Otherwise, yields the error and returns `next`

Use this after any operation that may fail:
```go
err := t.backup(pos)
return yieldErrorOr(err, tokenizeIdentifier)
```

### `yieldTokenThen(tok Token, next tokenizerAction) tokenizerAction`

Yields a token and continues with the next action:
```go
return yieldTokenThen(
    Token{Pos: pos, Type: TokenSymbol, Value: []byte{sym}},
    tokenizeIDL,
)
```

### `skipWhitespace(next tokenizerAction) tokenizerAction`

Wraps an action to skip leading whitespace before executing:
```go
return skipWhitespace(tokenizeIDL)
```

## Tokenizer Entry Point Pattern

The main `tokenizeIDL` function follows this structure:

```go
func tokenizeIDL(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
    return skipWhitespace(
        func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
            pos := t.pos
            r, err := t.next()

            return yieldErrorOr(
                err,
                func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
                    switch {
                    case r == '/':
                        return tokenizeComment(pos)
                    case isSymbol(r):
                        return tokenizeSymbol(pos, byte(r))
                    // ... more cases
                    }
                },
            )
        },
    )
}
```

Key pattern: Capture position before reading, then dispatch to specific tokenizer.

## Specific Tokenizer Functions

### Closure Pattern for Capturing State

When a tokenizer needs to capture state (like position), return a closure:

```go
func tokenizeComment(pos Pos) tokenizerAction {
    return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
        // pos is captured and available here
        r, err := t.next()
        return yieldErrorOr(
            err,
            func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
                switch r {
                case '/':
                    return tokenizeSingleLineComment(pos)
                case '*':
                    return tokenizeMultiLineComment(pos)
                default:
                    return yieldErrorOr(UnexpectedCharacterError{Pos: t.pos, R: r}, nil)
                }
            },
        )
    }
}
```

### Yielding Tokens

After successfully building a token, use `yieldTokenThen` chained with `skipWhitespace`:

```go
return yieldErrorOr(
    err,
    yieldTokenThen(
        Token{Pos: pos, Type: TokenComment, Value: comment.Bytes()},
        skipWhitespace(tokenizeIDL),
    ),
)
```

## Parser Pattern

### Iterator Usage

The parser uses `iter.Pull2` to convert the push-based tokenizer into pull-based:

```go
next, stop := iter.Pull2(Tokenize(r))
defer stop()

p := &parser{next: next}
```

### Expect Pattern

Use `p.expect()` to require specific token types:

```go
tok, err := p.expect(TokenIdentifier, TokenComment)
if err != nil {
    return nil, err
}
```

### Nested Parsing

When parsing nested structures, create a sub-loop:

```go
func parseSchema(p *parser, file *File) (_ parserAction[*File], err error) {
    for action := parseSchemaType; action != nil && err == nil; {
        action, err = action(p, file.Schema)
    }
    return nil, err
}
```

### Symbol Matching

For specific symbol values, check after expecting TokenSymbol:

```go
func parseSemicolon(next parserAction[*Schema]) parserAction[*Schema] {
    return func(p *parser, schema *Schema) (parserAction[*Schema], error) {
        tok, err := p.expect(TokenSymbol)
        if err != nil {
            return nil, err
        }
        if !bytes.Equal(tok.Value, []byte(";")) {
            return nil, UnexpectedTokenError{
                Expected: []TokenType{TokenSymbol},
                Actual:   tok,
            }
        }
        return next, nil
    }
}
```

## Printer Pattern

### Printer Actions

```go
type printerAction func(pr *printer, f *File) printerAction
```

- Each action takes a `*printer` and `*File`, returns the next action
- Return `nil` to end printing
- The printer accumulates errors in `pr.err`; actions short-circuit when an error is set

### Printer Helper Methods

#### `pr.write(s string)` / `pr.writef(format string, args ...any)`

Write output to the underlying writer. Both no-op if `pr.err` is already set:
```go
pr.write("schema ")
pr.writef("namespace %s;\n", f.Schema.Namespace)
```

### `writeThen(s string, next printerAction) printerAction`

Writes a string and returns the next action — the printer equivalent of `yieldTokenThen`:
```go
return writeThen(";", nil)
```

### Closure Pattern for Capturing State

Same closure pattern as the tokenizer — return a closure when state (like an index) needs to be captured:

```go
func printComments(idx int) printerAction {
    return func(pr *printer, f *File) printerAction {
        if idx >= len(f.Comments) {
            return printSchemaKeyword
        }
        pr.writef("%s\n", f.Comments[idx].Text)
        return printComments(idx + 1)
    }
}
```

### Type Dispatch

Use a type switch on the `Type` interface to determine how to print a schema type:

```go
func printType(t Type, next printerAction) printerAction {
    return func(pr *printer, f *File) printerAction {
        switch typ := t.(type) {
        case Ident:
            pr.write(typ.Value)
        }
        return next
    }
}
```

### Entry Point

The public `Print` function drives the action loop, checking `pr.err` each iteration:

```go
func Print(w io.Writer, f *File) error {
    pr := &printer{w: w}
    for action := printFile; action != nil && pr.err == nil; {
        action = action(pr, f)
    }
    return pr.err
}
```

## Testing Style for Printer

### Direct Print Tests

Test with an explicit `*File` input and an expected string output:

```go
{
    name: "primitive schema with custom namespace",
    input: &File{
        Schema: &Schema{
            Pos:       Pos{Line: 2, Column: 1},
            Namespace: "com.example",
            Type:      Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
        },
    },
    expected: `namespace com.example;
schema int;`,
},
```

### Round-Trip Tests

Validate Parse → Print → Parse produces equivalent ASTs. Compare semantic fields (namespace, comments, types) rather than positions:

```go
// Parse the original source
file1, err := Parse(strings.NewReader(tc.src))
require.NoError(t, err)

// Print the parsed AST
var buf bytes.Buffer
err = Print(&buf, file1)
require.NoError(t, err)

// Parse the printed output
file2, err := Parse(strings.NewReader(buf.String()))
require.NoError(t, err)

// Compare semantic fields, ignoring positions
require.Equal(t, file1.Schema.Namespace, file2.Schema.Namespace)
```

## Error Types

### Tokenizer Errors

```go
type UnexpectedCharacterError struct {
    Pos Pos
    R   rune
}
```

### Parser Errors

```go
type UnexpectedEndOfTokensError struct {
    Expected []TokenType
}

type UnexpectedTokenError struct {
    Expected []TokenType
    Actual   Token
}
```

## Testing Style for Tokenizer

Use `iter.Seq2` collection helper inside tests:

```go
collect := func(seq iter.Seq2[Token, error]) ([]Token, error) {
    tokens := make([]Token, 0, len(tc.expected))
    for item, err := range seq {
        if err != nil {
            return tokens, err
        }
        t.Log(item)
        tokens = append(tokens, item)
    }
    return tokens, nil
}

tokens, err := collect(Tokenize(strings.NewReader(tc.src)))

require.NoError(t, err)
require.Equal(t, tc.expected, tokens)
```

### Token Test Case Format

Specify exact positions for all tokens:

```go
{
    name: "primitive schema with default namespace",
    src:  `schema int;`,
    expected: []Token{
        {Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
        {Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
        {Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
    },
},
```
