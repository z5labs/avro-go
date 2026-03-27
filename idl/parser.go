// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"bytes"
	"errors"
	"io"
	"iter"
	"slices"
	"strings"
)

// Ident represents an identifier in the Avro IDL, such as a schema name,
// field name, or enum value name.
type Ident struct {
	Pos   Pos
	Value string
}

// SortOrder represents the sort order of a field in a record,
// which can be ascending, descending, or ignored.
type SortOrder int

const (
	SortOrderAsc SortOrder = iota
	SortOrderDesc
	SortOrderIgnore
)

// Field represents a field in a record.
type Field struct {
	Name      string
	Aliases   []string
	Type      Type
	SortOrder SortOrder

	// TODO: add support for default values of fields, which can be any valid
	// Avro JSON value, including null, boolean, number, string, array, and
	// object.
	//
	// Default   any
}

// Type represents a type in the Avro IDL.
type Type interface {
	idl()
}

// Record represents a record in the Avro IDL.
type Record struct {
	Name      string
	Namespace string
	Aliases   []string
	Fields    []*Field
}

func (Record) idl() {}

// Enum represents an enum in the Avro IDL.
type Enum struct {
	Name      string
	Namespace string
	Aliases   []string
	Values    []*Ident
	Default   *Ident
}

func (Enum) idl() {}

// Array represents an array in the Avro IDL.
type Array struct {
	Items Type
}

func (Array) idl() {}

// Map represents a map in the Avro IDL.
type Map struct {
	Values *Ident
}

func (Map) idl() {}

// Union represents a union in the Avro IDL.
type Union struct {
	Types []Type
}

func (Union) idl() {}

// Fixed represents a fixed in the Avro IDL.
type Fixed struct {
	Name      string
	Namespace string
	Aliases   []string
	Size      int
}

func (Fixed) idl() {}

// Schema represents a schema in the Avro IDL.
type Schema struct {
	Pos       Pos
	Namespace string
	Type      Ident
	Types     []Type
}

// Comment represents a comment in the Avro IDL.
type Comment struct {
	Pos  Pos
	Text string
}

// Protocol represents a protocol in the Avro IDL.
type Protocol struct {
	Pos       Pos
	Namespace string
}

// File represents a parsed Avro IDL file. Each Avro IDL file defines either a
// single Avro Protocol, or an Avro Schema with supporting named schemata in a
// namespace.
type File struct {
	Comments []*Comment
	Schema   *Schema
	Protocol *Protocol
}

// UnexpectedEndOfTokensError is the error returned by the parser when it reaches the end of the tokens unexpectedly.
type UnexpectedEndOfTokensError struct {
	Expected []TokenType
}

// Error implements the [error] interface.
func (e UnexpectedEndOfTokensError) Error() string {
	var expected []string
	for _, t := range e.Expected {
		expected = append(expected, t.String())
	}
	return "unexpected end of tokens, expected one of: " + strings.Join(expected, ", ")
}

// UnexpectedTokenError is the error returned by the parser when it encounters an unexpected token.
type UnexpectedTokenError struct {
	Expected []TokenType
	Actual   Token
}

// Error implements the [error] interface.
func (e UnexpectedTokenError) Error() string {
	var expected []string
	for _, t := range e.Expected {
		expected = append(expected, t.String())
	}
	return "unexpected token: " + e.Actual.String() + ", expected one of: " + strings.Join(expected, ", ")
}

// Parse the Avro IDL defined in the given reader.
func Parse(r io.Reader) (file *File, err error) {
	next, stop := iter.Pull2(Tokenize(r))
	defer stop()

	file = &File{}

	p := &parser{next: next}

	for action := parseFile; action != nil && err == nil; {
		action, err = action(p, file)
	}

	return
}

type parser struct {
	next func() (Token, error, bool)
}

func (p *parser) expect(expected ...TokenType) (Token, error) {
	tok, err, ok := p.next()
	if err != nil {
		return Token{}, err
	}
	if !ok {
		return Token{}, UnexpectedEndOfTokensError{Expected: expected}
	}

	if slices.Contains(expected, tok.Type) {
		return tok, nil
	}

	return Token{}, UnexpectedTokenError{
		Expected: expected,
		Actual:   tok,
	}
}

type parserAction[T any] func(p *parser, t T) (parserAction[T], error)

func parseFile(p *parser, file *File) (parserAction[*File], error) {
	tok, err := p.expect(TokenIdentifier, TokenComment)
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case TokenIdentifier:
		switch string(tok.Value) {
		case "schema":
			file.Schema = &Schema{Pos: tok.Pos}
			return parseSchema, nil
		case "namespace":
			file.Schema = &Schema{Pos: tok.Pos, Namespace: string(tok.Value)}
			return parseSchema, nil
		default:
			return nil, errors.New("schema idl must start with either 'schema' or 'namespace'")
		}
	case TokenComment:
		file.Comments = append(file.Comments, &Comment{
			Pos:  tok.Pos,
			Text: string(tok.Value),
		})

		return parseFile, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenComment},
			Actual:   tok,
		}
	}
}

func parseSchema(p *parser, file *File) (_ parserAction[*File], err error) {
	for action := parseSchemaType; action != nil && err == nil; {
		action, err = action(p, file.Schema)
	}

	return nil, err
}

func parseSchemaType(p *parser, schema *Schema) (parserAction[*Schema], error) {
	tok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}

	schema.Type = Ident{
		Pos:   tok.Pos,
		Value: string(tok.Value),
	}

	return parseSemicolon(parseTypes), nil
}

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

func parseTypes(p *parser, schema *Schema) (parserAction[*Schema], error) {
	_, err := p.expect(TokenIdentifier, TokenSymbol)
	if err != nil {
		var ueot UnexpectedEndOfTokensError
		if errors.As(err, &ueot) {
			// a schema can have zero types, so if we reach the end of tokens here, we can just return successfully.
			return nil, nil
		}

		return nil, err
	}

	return nil, nil
}
