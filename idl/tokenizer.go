// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"errors"
	"io"
	"iter"
)

// Pos represents the position of a token in the input.
type Pos struct {
	Line   int
	Column int
}

// Token represents a token in the Avro IDL.
type Token struct {
	Pos   Pos
	Type  TokenType
	Value []byte
}

// TokenType represents the type of a token.
type TokenType int

const (
	TokenComment      TokenType = iota // e.g. // comment or /* comment */
	TokenIdentifier                    // e.g. schema, enum, record, namespace, etc.
	TokenSemicolon                     // ;
	TokenAt                            // @
	TokenString                        // e.g. "string"
	TokenNumber                        // e.g. 123, 45.67
	TokenLeftBrace                     // {
	TokenRightBrace                    // }
	TokenLeftParen                     // (
	TokenRightParen                    // )
	TokenLeftBracket                   // [
	TokenRightBracket                  // ]
	TokenLessThan                      // <
	TokenGreaterThan                   // >
	TokenComma                         // ,
	TokenEqual                         // =
	TokenQuestion                      // ?
	TokenBacktick                      // `
)

// ErrEndOfTokens is the error returned by the tokenizer when it reaches the end of the input.
var ErrEndOfTokens = errors.New("idl: end of tokens")

// Tokenize the Avro IDL defined in the given reader.
func Tokenize(r io.Reader) iter.Seq2[Token, error] {
	return func(yield func(Token, error) bool) {}
}
