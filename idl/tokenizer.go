// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"unicode"
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

func (t Token) String() string {
	return fmt.Sprintf("%s(%s)", t.Type, t.Value)
}

// TokenType represents the type of a token.
type TokenType int

const (
	TokenComment    TokenType = iota // e.g. // comment or /* comment */
	TokenDocComment                  // e.g. /** doc comment */
	TokenIdentifier                  // e.g. schema, enum, record, namespace, etc.
	TokenSymbol                      // e.g. ";", "<", ">", "{", "}", "(", ")", "[", "]", ",", "=", "?", "`"
	TokenString                      // e.g. "string"
	TokenNumber                      // e.g. 123, 45.67
	TokenAnnotation                  // e.g. @namespace, @order, @java-class
)

func (tt TokenType) String() string {
	switch tt {
	case TokenComment:
		return "Comment"
	case TokenDocComment:
		return "DocComment"
	case TokenIdentifier:
		return "Identifier"
	case TokenSymbol:
		return "Symbol"
	case TokenString:
		return "String"
	case TokenNumber:
		return "Number"
	case TokenAnnotation:
		return "Annotation"
	default:
		panic(fmt.Sprintf("unknown token type: %d", tt))
	}
}

// Tokenize the Avro IDL defined in the given reader.
func Tokenize(r io.Reader) iter.Seq2[Token, error] {
	return func(yield func(Token, error) bool) {
		t := &tokenizer{
			pos: Pos{Line: 1, Column: 1},
			buf: bufio.NewReader(r),
		}

		for action := tokenizeIDL; action != nil; {
			action = action(t, yield)
		}
	}
}

type tokenizer struct {
	// pos tracks the current position in the input for error reporting.
	pos Pos

	buf *bufio.Reader
}

func (t *tokenizer) next() (rune, error) {
	r, size, err := t.buf.ReadRune()
	if err != nil {
		return 0, err
	}
	t.pos.Column += size
	if r == '\n' {
		t.pos.Line++
		t.pos.Column = 1
	}
	return r, nil
}

func (t *tokenizer) backup(previousPos Pos) error {
	err := t.buf.UnreadRune()
	if err != nil {
		return err
	}
	t.pos = previousPos
	return nil
}

func (t *tokenizer) copyIf(buf *bytes.Buffer, cond func(rune) bool) error {
	for {
		r, size, err := t.buf.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return io.ErrUnexpectedEOF
			}
			return err
		}

		if !cond(r) {
			err = t.buf.UnreadRune()
			return err
		}

		_, err = buf.WriteRune(r)
		if err != nil {
			return err
		}

		t.pos.Column += size
		if r == '\n' {
			t.pos.Line++
			t.pos.Column = 1
		}
	}
}

func (t *tokenizer) copyUntil(dst *bytes.Buffer, delim []rune) error {
	buf := make([]rune, 0, len(delim))

	for {
		r, size, err := t.buf.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return io.ErrUnexpectedEOF
			}
			return err
		}

		if len(buf) == len(delim) {
			popRune := buf[0]
			buf = buf[1:]

			_, err := dst.WriteRune(popRune)
			if err != nil {
				return err
			}
		}

		buf = append(buf, r)

		if slices.Equal(buf, delim) {
			return nil
		}

		t.pos.Column += size
		if r == '\n' {
			t.pos.Line++
			t.pos.Column = 1
		}
	}
}

type tokenizerAction func(t *tokenizer, yield func(Token, error) bool) tokenizerAction

func yieldErrorOr(err error, next tokenizerAction) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		if err == nil {
			return next
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		if !yield(Token{}, err) {
			return nil
		}
		return next
	}
}

func yieldTokenThen(tok Token, next tokenizerAction) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		if !yield(tok, nil) {
			return nil
		}
		return next
	}
}

func skipWhitespace(next tokenizerAction) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		var buf bytes.Buffer
		err := t.copyIf(&buf, unicode.IsSpace)

		return yieldErrorOr(err, next)
	}
}

// UnexpectedCharacterError is the error returned by the tokenizer when it encounters an unexpected character in the input.
type UnexpectedCharacterError struct {
	Pos Pos
	R   rune
}

// Error implements the [error] interface.
func (e UnexpectedCharacterError) Error() string {
	return fmt.Sprintf("unexpected character '%c' at line %d, column %d", e.R, e.Pos.Line, e.Pos.Column)
}

// UnterminatedStringError is the error returned by the tokenizer when it encounters an unterminated string literal.
type UnterminatedStringError struct {
	Pos Pos
}

// Error implements the [error] interface.
func (e UnterminatedStringError) Error() string {
	return fmt.Sprintf("unterminated string literal at line %d, column %d", e.Pos.Line, e.Pos.Column)
}

// InvalidNumberError is the error returned by the tokenizer when it encounters an invalid number literal.
type InvalidNumberError struct {
	Pos   Pos
	Value string
}

// Error implements the [error] interface.
func (e InvalidNumberError) Error() string {
	return fmt.Sprintf("invalid number literal %q at line %d, column %d", e.Value, e.Pos.Line, e.Pos.Column)
}

func tokenizeIDL(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
	return skipWhitespace(
		func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
			pos := t.pos
			r, err := t.next()

			return yieldErrorOr(
				err,
				func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
					switch {
					case r == '@':
						return tokenizeAnnotation(pos)
					case r == '/':
						return tokenizeComment(pos)
					case r == '"':
						return tokenizeString(pos)
					case isSymbol(r):
						return tokenizeSymbol(pos, byte(r))
					case unicode.IsLetter(r) || r == '_':
						err = t.backup(pos)
						return yieldErrorOr(
							err,
							tokenizeIdentifier,
						)
					case r == '-' || unicode.IsDigit(r):
						err = t.backup(pos)
						return yieldErrorOr(
							err,
							tokenizeNumber,
						)
					default:
						return yieldErrorOr(UnexpectedCharacterError{Pos: t.pos, R: r}, nil)
					}
				},
			)
		},
	)
}

func isSymbol(r rune) bool {
	switch r {
	case ';', '{', '}', '(', ')', '[', ']', '<', '>', ',', '=', '?', '`', ':':
		return true
	default:
		return false
	}
}

func tokenizeComment(pos Pos) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
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

func tokenizeSingleLineComment(pos Pos) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		var comment bytes.Buffer
		comment.Write([]byte{'/', '/'})

		err := t.copyIf(&comment, func(r rune) bool {
			return r != '\n'
		})

		return yieldErrorOr(
			err,
			yieldTokenThen(
				Token{Pos: pos, Type: TokenComment, Value: comment.Bytes()},
				skipWhitespace(tokenizeIDL),
			),
		)
	}
}

func tokenizeMultiLineComment(pos Pos) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		var comment bytes.Buffer
		comment.Write([]byte{'/', '*'})

		err := t.copyUntil(&comment, []rune{'*', '/'})
		comment.Write([]byte{'*', '/'})

		// Check if this is a doc comment (starts with /** and is longer than /**/)
		tokenType := TokenComment
		val := comment.Bytes()
		if len(val) > 4 && val[2] == '*' {
			tokenType = TokenDocComment
		}

		return yieldErrorOr(
			err,
			yieldTokenThen(
				Token{Pos: pos, Type: tokenType, Value: val},
				skipWhitespace(tokenizeIDL),
			),
		)
	}
}

func tokenizeIdentifier(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
	pos := t.pos

	var ident bytes.Buffer
	err := t.copyIf(&ident, func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
	})

	tok := Token{Pos: pos, Type: TokenIdentifier, Value: ident.Bytes()}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return yieldTokenThen(tok, nil)
	}

	return yieldErrorOr(
		err,
		yieldTokenThen(tok, skipWhitespace(tokenizeIDL)),
	)
}

func tokenizeNumber(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
	pos := t.pos

	first := true
	hasDot := false
	var num bytes.Buffer
	err := t.copyIf(&num, func(r rune) bool {
		if first {
			first = false
			return r == '-' || unicode.IsDigit(r)
		}
		if r == '.' && !hasDot {
			hasDot = true
			return true
		}
		return unicode.IsDigit(r)
	})

	if err == nil || errors.Is(err, io.ErrUnexpectedEOF) {
		s := num.String()
		valid := true
		switch {
		case s == "-" || s == ".":
			valid = false
		case len(s) > 1 && s[len(s)-1] == '.':
			valid = false
		case len(s) > 1 && s[0] == '-' && s[1] == '.':
			valid = false
		}
		if !valid {
			return yieldErrorOr(InvalidNumberError{Pos: pos, Value: s}, nil)
		}
	}

	tok := Token{Pos: pos, Type: TokenNumber, Value: num.Bytes()}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return yieldTokenThen(tok, nil)
	}

	return yieldErrorOr(
		err,
		yieldTokenThen(tok, skipWhitespace(tokenizeIDL)),
	)
}

func tokenizeString(pos Pos) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		var str bytes.Buffer
		escaped := false
		for {
			r, err := t.next()
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					if !yield(Token{}, UnterminatedStringError{Pos: pos}) {
						return nil
					}
					return nil
				}
				return yieldErrorOr(err, nil)
			}
			if escaped {
				str.WriteRune('\\')
				str.WriteRune(r)
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				tok := Token{Pos: pos, Type: TokenString, Value: str.Bytes()}
				return yieldTokenThen(tok, skipWhitespace(tokenizeIDL))
			}
			str.WriteRune(r)
		}
	}
}

// InvalidAnnotationError is the error returned by the tokenizer when it encounters an annotation with an empty or invalid name.
type InvalidAnnotationError struct {
	Pos Pos
}

// Error implements the [error] interface.
func (e InvalidAnnotationError) Error() string {
	return fmt.Sprintf("invalid annotation at line %d, column %d: expected name after '@'", e.Pos.Line, e.Pos.Column)
}

func tokenizeAnnotation(pos Pos) tokenizerAction {
	return func(t *tokenizer, yield func(Token, error) bool) tokenizerAction {
		var name bytes.Buffer
		err := t.copyIf(&name, func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '-'
		})

		if name.Len() == 0 {
			return yieldErrorOr(InvalidAnnotationError{Pos: pos}, nil)
		}

		tok := Token{Pos: pos, Type: TokenAnnotation, Value: name.Bytes()}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return yieldTokenThen(tok, nil)
		}

		return yieldErrorOr(
			err,
			yieldTokenThen(tok, skipWhitespace(tokenizeIDL)),
		)
	}
}

func tokenizeSymbol(pos Pos, sym byte) tokenizerAction {
	return yieldTokenThen(
		Token{Pos: pos, Type: TokenSymbol, Value: []byte{sym}},
		tokenizeIDL,
	)
}
