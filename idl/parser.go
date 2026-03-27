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
	"strconv"
	"strings"
)

// Ident represents an identifier in the Avro IDL, such as a schema name,
// field name, or enum value name.
type Ident struct {
	Pos   Pos
	Value string
}

func (Ident) idl() {}

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
	Type      Type
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

			return parseSchemaType(func(typ Type) (parserAction[*File], error) {
				file.Schema.Type = typ
				return parseOptionalNullable(&file.Schema.Type, parseSchemaTypes), nil
			}), nil
		case "namespace":
			file.Schema = &Schema{}

			return parseIdent(func(t Token) (parserAction[*File], error) {
				file.Schema.Namespace = string(t.Value)
				return parseSemicolon(parseSchema), nil
			}), nil
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

func parseSchema(p *parser, file *File) (parserAction[*File], error) {
	tok, err := p.expect(TokenIdentifier, TokenComment)
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case TokenIdentifier:
		switch string(tok.Value) {
		case "schema":
			file.Schema.Pos = tok.Pos
			return parseSchemaType(func(typ Type) (parserAction[*File], error) {
				file.Schema.Type = typ
				return parseOptionalNullable(&file.Schema.Type, parseSchemaTypes), nil
			}), nil
		default:
			return nil, errors.New("schema definition must follow namespace declaration")
		}
	case TokenComment:
		file.Comments = append(file.Comments, &Comment{
			Pos:  tok.Pos,
			Text: string(tok.Value),
		})
		return parseSchema, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenComment},
			Actual:   tok,
		}
	}
}

func parseSchemaTypes(p *parser, file *File) (_ parserAction[*File], err error) {
	for action := parseType; action != nil && err == nil; {
		action, err = action(p, file.Schema)
	}

	return nil, err
}

func parseIdent[T any](f func(Token) (parserAction[T], error)) parserAction[T] {
	return func(p *parser, t T) (parserAction[T], error) {
		tok, err := p.expect(TokenIdentifier)
		if err != nil {
			return nil, err
		}

		return f(tok)
	}
}

func parseSemicolon[T any](next parserAction[T]) parserAction[T] {
	return func(p *parser, t T) (parserAction[T], error) {
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

func parseOptionalNullable[T any](typePtr *Type, next parserAction[T]) parserAction[T] {
	return func(p *parser, t T) (parserAction[T], error) {
		tok, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		switch {
		case bytes.Equal(tok.Value, []byte("?")):
			*typePtr = &Union{Types: []Type{Ident{Value: "null"}, *typePtr}}
			return parseSemicolon(next), nil
		case bytes.Equal(tok.Value, []byte(";")):
			return next, nil
		default:
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   tok,
			}
		}
	}
}

func parseSchemaType[T any](f func(Type) (parserAction[T], error)) parserAction[T] {
	return func(p *parser, t T) (parserAction[T], error) {
		typ, err := parseTypeRef(p)
		if err != nil {
			return nil, err
		}
		return f(typ)
	}
}

func parseTypeRef(p *parser) (Type, error) {
	tok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	switch string(tok.Value) {
	case "map":
		return parseMapType(p)
	case "union":
		return parseUnionType(p)
	default:
		return Ident{Pos: tok.Pos, Value: string(tok.Value)}, nil
	}
}

func parseMapType(p *parser) (*Map, error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(tok.Value, []byte("<")) {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}

	valTok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}

	tok, err = p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(tok.Value, []byte(">")) {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}

	return &Map{
		Values: &Ident{Pos: valTok.Pos, Value: string(valTok.Value)},
	}, nil
}

func parseUnionType(p *parser) (u *Union, err error) {
	u = &Union{}
	for action := parseUnionOpenBrace; action != nil && err == nil; {
		action, err = action(p, u)
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func parseUnionOpenBrace(p *parser, u *Union) (parserAction[*Union], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(tok.Value, []byte("{")) {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
	return parseUnionMember, nil
}

func parseUnionMember(p *parser, u *Union) (parserAction[*Union], error) {
	typ, err := parseTypeRef(p)
	if err != nil {
		return nil, err
	}
	u.Types = append(u.Types, typ)
	return parseUnionMemberSep, nil
}

func parseUnionMemberSep(p *parser, u *Union) (parserAction[*Union], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	switch {
	case bytes.Equal(tok.Value, []byte(",")):
		return parseUnionMemberOrClose, nil
	case bytes.Equal(tok.Value, []byte("}")):
		return nil, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseUnionMemberOrClose(p *parser, u *Union) (parserAction[*Union], error) {
	tok, err := p.expect(TokenIdentifier, TokenSymbol)
	if err != nil {
		return nil, err
	}
	switch tok.Type {
	case TokenIdentifier:
		switch string(tok.Value) {
		case "map":
			m, err := parseMapType(p)
			if err != nil {
				return nil, err
			}
			u.Types = append(u.Types, m)
		case "union":
			nested, err := parseUnionType(p)
			if err != nil {
				return nil, err
			}
			u.Types = append(u.Types, nested)
		default:
			u.Types = append(u.Types, Ident{Pos: tok.Pos, Value: string(tok.Value)})
		}
		return parseUnionMemberSep, nil
	case TokenSymbol:
		if !bytes.Equal(tok.Value, []byte("}")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   tok,
			}
		}
		return nil, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseType(p *parser, schema *Schema) (_ parserAction[*Schema], err error) {
	tok, err, ok := p.next()
	if !ok {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case TokenIdentifier:
		switch string(tok.Value) {
		case "enum":
			enum, err := parseEnum(p)
			if err != nil {
				return nil, err
			}
			schema.Types = append(schema.Types, enum)
			return parseType, nil
		case "fixed":
			fixed, err := parseFixed(p)
			if err != nil {
				return nil, err
			}
			schema.Types = append(schema.Types, fixed)
			return parseType, nil
		default:
			return nil, errors.New("unknown type keyword: " + string(tok.Value))
		}
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier},
			Actual:   tok,
		}
	}
}

func parseEnum(p *parser) (enum *Enum, err error) {
	enum = &Enum{}
	for action := parseEnumName(enum); action != nil && err == nil; {
		action, err = action(p, enum)
	}
	if err != nil {
		return nil, err
	}
	return enum, nil
}

func parseEnumName(enum *Enum) parserAction[*Enum] {
	return parseIdent(func(tok Token) (parserAction[*Enum], error) {
		enum.Name = string(tok.Value)
		return parseEnumOpenBrace, nil
	})
}

func parseEnumOpenBrace(p *parser, enum *Enum) (parserAction[*Enum], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(tok.Value, []byte("{")) {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
	return parseEnumValue, nil
}

func parseEnumValue(p *parser, enum *Enum) (parserAction[*Enum], error) {
	tok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	enum.Values = append(enum.Values, &Ident{
		Pos:   tok.Pos,
		Value: string(tok.Value),
	})
	return parseEnumValueSep, nil
}

func parseEnumValueSep(p *parser, enum *Enum) (parserAction[*Enum], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	switch {
	case bytes.Equal(tok.Value, []byte(",")):
		return parseEnumValueOrClose, nil
	case bytes.Equal(tok.Value, []byte("}")):
		return parseEnumDefaultOrSemicolon, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseEnumValueOrClose(p *parser, enum *Enum) (parserAction[*Enum], error) {
	tok, err := p.expect(TokenIdentifier, TokenSymbol)
	if err != nil {
		return nil, err
	}
	switch tok.Type {
	case TokenIdentifier:
		enum.Values = append(enum.Values, &Ident{
			Pos:   tok.Pos,
			Value: string(tok.Value),
		})
		return parseEnumValueSep, nil
	case TokenSymbol:
		if !bytes.Equal(tok.Value, []byte("}")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   tok,
			}
		}
		return parseEnumDefaultOrSemicolon, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseEnumDefaultOrSemicolon(p *parser, enum *Enum) (parserAction[*Enum], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	switch {
	case bytes.Equal(tok.Value, []byte("=")):
		return parseEnumDefault, nil
	case bytes.Equal(tok.Value, []byte(";")):
		return nil, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseEnumDefault(p *parser, enum *Enum) (parserAction[*Enum], error) {
	tok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	enum.Default = &Ident{
		Pos:   tok.Pos,
		Value: string(tok.Value),
	}
	return parseSemicolon[*Enum](nil), nil
}

func parseFixed(p *parser) (fixed *Fixed, err error) {
	fixed = &Fixed{}
	for action := parseFixedName(fixed); action != nil && err == nil; {
		action, err = action(p, fixed)
	}
	if err != nil {
		return nil, err
	}
	return fixed, nil
}

func parseFixedName(fixed *Fixed) parserAction[*Fixed] {
	return parseIdent(func(tok Token) (parserAction[*Fixed], error) {
		fixed.Name = string(tok.Value)
		return parseFixedOpenParen, nil
	})
}

func parseFixedOpenParen(p *parser, fixed *Fixed) (parserAction[*Fixed], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(tok.Value, []byte("(")) {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
	return parseFixedSize, nil
}

func parseFixedSize(p *parser, fixed *Fixed) (parserAction[*Fixed], error) {
	tok, err := p.expect(TokenNumber)
	if err != nil {
		return nil, err
	}
	size, err := strconv.Atoi(string(tok.Value))
	if err != nil {
		return nil, err
	}
	fixed.Size = size
	return parseFixedCloseParen, nil
}

func parseFixedCloseParen(p *parser, fixed *Fixed) (parserAction[*Fixed], error) {
	tok, err := p.expect(TokenSymbol)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(tok.Value, []byte(")")) {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
	return parseSemicolon[*Fixed](nil), nil
}
