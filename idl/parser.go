// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"bytes"
	"errors"
	"fmt"
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
	Doc        string
	Name       string
	Aliases    []string
	Type       Type
	SortOrder  SortOrder
	Properties map[string]Value

	// A nil Value means no default was specified.
	Default Value
}

// Value represents a value.
type Value interface {
	val()
}

// NullValue represents a JSON null value.
type NullValue struct{}

func (NullValue) val() {}

// BoolValue represents a JSON boolean value.
type BoolValue bool

func (BoolValue) val() {}

// IntValue represents a JSON integer value.
type IntValue int64

func (IntValue) val() {}

// FloatValue represents a JSON floating-point value.
type FloatValue float64

func (FloatValue) val() {}

// StringValue represents a JSON string value.
type StringValue string

func (StringValue) val() {}

// ArrayValue represents a JSON array value.
type ArrayValue []Value

func (ArrayValue) val() {}

// ObjectValue represents a JSON object value.
type ObjectValue map[string]Value

func (ObjectValue) val() {}

// Annotation represents a parsed annotation from the Avro IDL (e.g. @namespace("org.example")).
type Annotation struct {
	Name  string
	Value Value
}

// Type represents a type in the Avro IDL.
type Type interface {
	idl()
}

// Record represents a record in the Avro IDL.
type Record struct {
	Doc        string
	Name       string
	Namespace  string
	Aliases    []string
	Fields     []*Field
	Properties map[string]Value
}

func (Record) idl() {}

// Enum represents an enum in the Avro IDL.
type Enum struct {
	Doc        string
	Name       string
	Namespace  string
	Aliases    []string
	Values     []*Ident
	Default    *Ident
	Properties map[string]Value
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
	Doc        string
	Name       string
	Namespace  string
	Aliases    []string
	Size       int
	Properties map[string]Value
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

// UnterminatedEscapedIdentifierError is the error returned by the parser when it encounters an
// escaped identifier that is missing a closing backtick.
type UnterminatedEscapedIdentifierError struct {
	Pos Pos
}

// Error implements the [error] interface.
func (e UnterminatedEscapedIdentifierError) Error() string {
	return fmt.Sprintf("unterminated escaped identifier at line %d, column %d", e.Pos.Line, e.Pos.Column)
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
	next    func() (Token, error, bool)
	pending *Token
}

func (p *parser) unread(tok Token) {
	p.pending = &tok
}

func (p *parser) read() (Token, error, bool) {
	if p.pending != nil {
		tok := *p.pending
		p.pending = nil
		return tok, nil, true
	}
	return p.next()
}

func (p *parser) peek() (Token, error, bool) {
	if p.pending != nil {
		return *p.pending, nil, true
	}
	tok, err, ok := p.next()
	if err != nil {
		return Token{}, err, false
	}
	if !ok {
		return Token{}, nil, false
	}
	p.pending = &tok
	return tok, nil, true
}

func (p *parser) expect(expected ...TokenType) (Token, error) {
	tok, err, ok := p.read()
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

// expectIdentifier reads either a regular identifier token or an escaped
// identifier enclosed in backticks. For escaped identifiers like `error`,
// it returns a Token containing the unescaped value ("error").
func (p *parser) expectIdentifier() (Token, error) {
	tok, err, ok := p.read()
	if err != nil {
		return Token{}, err
	}
	if !ok {
		return Token{}, UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier}}
	}

	// Regular identifier
	if tok.Type == TokenIdentifier {
		return tok, nil
	}

	// Escaped identifier: ` identifier `
	if tok.Type == TokenSymbol && bytes.Equal(tok.Value, []byte("`")) {
		startPos := tok.Pos
		identTok, err := p.expect(TokenIdentifier)
		if err != nil {
			return Token{}, err
		}

		closeTok, err, ok := p.read()
		if err != nil {
			return Token{}, err
		}
		if !ok {
			return Token{}, UnterminatedEscapedIdentifierError{Pos: startPos}
		}
		if closeTok.Type != TokenSymbol || !bytes.Equal(closeTok.Value, []byte("`")) {
			return Token{}, UnterminatedEscapedIdentifierError{Pos: startPos}
		}

		return Token{Pos: startPos, Type: TokenIdentifier, Value: identTok.Value}, nil
	}

	return Token{}, UnexpectedTokenError{
		Expected: []TokenType{TokenIdentifier},
		Actual:   tok,
	}
}

type parserAction[T any] func(p *parser, t T) (parserAction[T], error)

func cleanDocComment(raw string) string {
	// Strip /** prefix and */ suffix
	s := strings.TrimPrefix(raw, "/**")
	s = strings.TrimSuffix(s, "*/")

	// Split into lines and clean each line
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		// Trim leading whitespace
		line = strings.TrimLeft(line, " \t")
		// Strip leading * if present
		line = strings.TrimPrefix(line, "*")
		// Trim a single leading space after * if present
		if len(line) > 0 && line[0] == ' ' {
			line = line[1:]
		}
		lines[i] = line
	}

	// Join and trim
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func skipComments(p *parser) error {
	for {
		tok, err, ok := p.peek()
		if err != nil {
			return err
		}
		if !ok || (tok.Type != TokenComment && tok.Type != TokenDocComment) {
			return nil
		}
		p.pending = nil
	}
}

func collectDocComment(p *parser) (string, error) {
	tok, err, ok := p.peek()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	if tok.Type != TokenDocComment {
		return "", nil
	}
	// Consume the peeked doc comment token
	p.pending = nil
	return cleanDocComment(string(tok.Value)), nil
}

func collectAnnotations(p *parser) ([]*Annotation, error) {
	var annotations []*Annotation
	for {
		tok, err, ok := p.peek()
		if err != nil {
			return nil, err
		}
		if !ok {
			return annotations, nil
		}
		if tok.Type != TokenAnnotation {
			return annotations, nil
		}
		// Consume the peeked annotation token.
		p.pending = nil

		// Read "("
		open, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(open.Value, []byte("(")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   open,
			}
		}

		// Read value
		val, err := parseJSONValue(p)
		if err != nil {
			return nil, err
		}

		// Read ")"
		close, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(close.Value, []byte(")")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   close,
			}
		}

		annotations = append(annotations, &Annotation{
			Name:  string(tok.Value),
			Value: val,
		})
	}
}

func applyAnnotationsToRecord(rec *Record, annotations []*Annotation) {
	for _, ann := range annotations {
		switch ann.Name {
		case "namespace":
			if sv, ok := ann.Value.(StringValue); ok {
				rec.Namespace = string(sv)
			}
		case "aliases":
			rec.Aliases = extractStringSlice(ann.Value)
		default:
			if rec.Properties == nil {
				rec.Properties = make(map[string]Value)
			}
			rec.Properties[ann.Name] = ann.Value
		}
	}
}

func applyAnnotationsToEnum(enum *Enum, annotations []*Annotation) {
	for _, ann := range annotations {
		switch ann.Name {
		case "namespace":
			if sv, ok := ann.Value.(StringValue); ok {
				enum.Namespace = string(sv)
			}
		case "aliases":
			enum.Aliases = extractStringSlice(ann.Value)
		default:
			if enum.Properties == nil {
				enum.Properties = make(map[string]Value)
			}
			enum.Properties[ann.Name] = ann.Value
		}
	}
}

func applyAnnotationsToFixed(fixed *Fixed, annotations []*Annotation) {
	for _, ann := range annotations {
		switch ann.Name {
		case "namespace":
			if sv, ok := ann.Value.(StringValue); ok {
				fixed.Namespace = string(sv)
			}
		case "aliases":
			fixed.Aliases = extractStringSlice(ann.Value)
		default:
			if fixed.Properties == nil {
				fixed.Properties = make(map[string]Value)
			}
			fixed.Properties[ann.Name] = ann.Value
		}
	}
}

func applyAnnotationsToField(field *Field, annotations []*Annotation) {
	for _, ann := range annotations {
		switch ann.Name {
		case "order":
			if sv, ok := ann.Value.(StringValue); ok {
				switch string(sv) {
				case "ascending":
					field.SortOrder = SortOrderAsc
				case "descending":
					field.SortOrder = SortOrderDesc
				case "ignore":
					field.SortOrder = SortOrderIgnore
				}
			}
		case "aliases":
			field.Aliases = extractStringSlice(ann.Value)
		default:
			if field.Properties == nil {
				field.Properties = make(map[string]Value)
			}
			field.Properties[ann.Name] = ann.Value
		}
	}
}

func extractStringSlice(v Value) []string {
	arr, ok := v.(ArrayValue)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, elem := range arr {
		if sv, ok := elem.(StringValue); ok {
			result = append(result, string(sv))
		}
	}
	return result
}

func parseFile(p *parser, file *File) (parserAction[*File], error) {
	tok, err := p.expect(TokenIdentifier, TokenComment, TokenDocComment)
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
	case TokenComment, TokenDocComment:
		file.Comments = append(file.Comments, &Comment{
			Pos:  tok.Pos,
			Text: string(tok.Value),
		})

		return parseFile, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenComment, TokenDocComment},
			Actual:   tok,
		}
	}
}

func parseSchema(p *parser, file *File) (parserAction[*File], error) {
	tok, err := p.expect(TokenIdentifier, TokenComment, TokenDocComment)
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
	case TokenComment, TokenDocComment:
		file.Comments = append(file.Comments, &Comment{
			Pos:  tok.Pos,
			Text: string(tok.Value),
		})
		return parseSchema, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenComment, TokenDocComment},
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
		tok, err := p.expectIdentifier()
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
	tok, err := p.expectIdentifier()
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

	valTok, err := p.expectIdentifier()
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
	tok, err, ok := p.read()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenSymbol}}
	}

	// Check for closing brace
	if tok.Type == TokenSymbol {
		if bytes.Equal(tok.Value, []byte("}")) {
			return nil, nil
		}
		// Check for escaped identifier starting with backtick
		if bytes.Equal(tok.Value, []byte("`")) {
			p.unread(tok)
			identTok, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			u.Types = append(u.Types, Ident{Pos: identTok.Pos, Value: string(identTok.Value)})
			return parseUnionMemberSep, nil
		}
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}

	if tok.Type != TokenIdentifier {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}

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
}

func parseType(p *parser, schema *Schema) (_ parserAction[*Schema], err error) {
	doc, err := collectDocComment(p)
	if err != nil {
		return nil, err
	}

	annotations, err := collectAnnotations(p)
	if err != nil {
		return nil, err
	}

	tok, err, ok := p.read()
	if !ok {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return dispatchType(tok, doc, annotations), nil
}

func dispatchType(tok Token, doc string, annotations []*Annotation) parserAction[*Schema] {
	return func(p *parser, schema *Schema) (_ parserAction[*Schema], err error) {
		switch tok.Type {
		case TokenIdentifier:
			switch string(tok.Value) {
			case "enum":
				enum, err := parseEnum(p)
				if err != nil {
					return nil, err
				}
				enum.Doc = doc
				applyAnnotationsToEnum(enum, annotations)
				schema.Types = append(schema.Types, enum)
				return parseEnumOptionalDefault(enum), nil
			case "fixed":
				fixed, err := parseFixed(p)
				if err != nil {
					return nil, err
				}
				fixed.Doc = doc
				applyAnnotationsToFixed(fixed, annotations)
				schema.Types = append(schema.Types, fixed)
				return parseType, nil
			case "record":
				rec, err := parseRecord(p)
				if err != nil {
					return nil, err
				}
				rec.Doc = doc
				applyAnnotationsToRecord(rec, annotations)
				schema.Types = append(schema.Types, rec)
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
}

func parseEnumOptionalDefault(enum *Enum) parserAction[*Schema] {
	return func(p *parser, schema *Schema) (parserAction[*Schema], error) {
		tok, err, ok := p.read()
		if !ok {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if tok.Type == TokenSymbol && bytes.Equal(tok.Value, []byte("=")) {
			defTok, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			enum.Default = &Ident{
				Pos:   defTok.Pos,
				Value: string(defTok.Value),
			}
			return parseSemicolon(parseType), nil
		}

		// The token might be a doc comment, annotation, or type keyword.
		var doc string
		if tok.Type == TokenDocComment {
			doc = cleanDocComment(string(tok.Value))
			tok, err, ok = p.read()
			if !ok {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
		}

		if tok.Type == TokenAnnotation {
			p.unread(tok)
			annotations, err := collectAnnotations(p)
			if err != nil {
				return nil, err
			}
			tok, err, ok = p.read()
			if !ok {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			return dispatchType(tok, doc, annotations), nil
		}
		return dispatchType(tok, doc, nil), nil
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
	if err := skipComments(p); err != nil {
		return nil, err
	}
	tok, err := p.expectIdentifier()
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
		return nil, nil
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseEnumValueOrClose(p *parser, enum *Enum) (parserAction[*Enum], error) {
	if err := skipComments(p); err != nil {
		return nil, err
	}
	tok, err, ok := p.read()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenSymbol}}
	}

	// Check for closing brace
	if tok.Type == TokenSymbol {
		if bytes.Equal(tok.Value, []byte("}")) {
			return nil, nil
		}
		// Check for escaped identifier starting with backtick
		if bytes.Equal(tok.Value, []byte("`")) {
			p.unread(tok)
			identTok, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			enum.Values = append(enum.Values, &Ident{
				Pos:   identTok.Pos,
				Value: string(identTok.Value),
			})
			return parseEnumValueSep, nil
		}
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}

	if tok.Type != TokenIdentifier {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}

	enum.Values = append(enum.Values, &Ident{
		Pos:   tok.Pos,
		Value: string(tok.Value),
	})
	return parseEnumValueSep, nil
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

func parseRecord(p *parser) (rec *Record, err error) {
	rec = &Record{}
	for action := parseRecordName(rec); action != nil && err == nil; {
		action, err = action(p, rec)
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func parseRecordName(rec *Record) parserAction[*Record] {
	return parseIdent(func(tok Token) (parserAction[*Record], error) {
		rec.Name = string(tok.Value)
		return parseRecordOpenBrace, nil
	})
}

func parseRecordOpenBrace(p *parser, rec *Record) (parserAction[*Record], error) {
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
	return parseRecordFieldType, nil
}

func parseRecordFieldType(p *parser, rec *Record) (parserAction[*Record], error) {
	doc, err := collectDocComment(p)
	if err != nil {
		return nil, err
	}
	preTypeAnnotations, err := collectAnnotations(p)
	if err != nil {
		return nil, err
	}
	typ, err := parseTypeRef(p)
	if err != nil {
		return nil, err
	}
	field := &Field{Doc: doc, Type: typ}
	applyAnnotationsToField(field, preTypeAnnotations)
	rec.Fields = append(rec.Fields, field)
	return parseRecordFieldNullableOrName(field), nil
}

func parseRecordFieldNullableOrName(field *Field) parserAction[*Record] {
	return func(p *parser, rec *Record) (parserAction[*Record], error) {
		tok, err, ok := p.read()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenSymbol, TokenAnnotation}}
		}

		switch tok.Type {
		case TokenAnnotation:
			p.unread(tok)
			annotations, err := collectAnnotations(p)
			if err != nil {
				return nil, err
			}
			applyAnnotationsToField(field, annotations)
			return parseRecordFieldName(field), nil
		case TokenIdentifier:
			field.Name = string(tok.Value)
			return parseRecordFieldDefaultOrSemicolon(field), nil
		case TokenSymbol:
			if bytes.Equal(tok.Value, []byte("?")) {
				field.Type = &Union{Types: []Type{Ident{Value: "null"}, field.Type}}
				return parseRecordFieldAnnotationsOrName(field), nil
			}
			// Check for escaped identifier starting with backtick
			if bytes.Equal(tok.Value, []byte("`")) {
				p.unread(tok)
				identTok, err := p.expectIdentifier()
				if err != nil {
					return nil, err
				}
				field.Name = string(identTok.Value)
				return parseRecordFieldDefaultOrSemicolon(field), nil
			}
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier, TokenSymbol, TokenAnnotation},
				Actual:   tok,
			}
		default:
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier, TokenSymbol, TokenAnnotation},
				Actual:   tok,
			}
		}
	}
}

func parseRecordFieldAnnotationsOrName(field *Field) parserAction[*Record] {
	return func(p *parser, rec *Record) (parserAction[*Record], error) {
		annotations, err := collectAnnotations(p)
		if err != nil {
			return nil, err
		}
		applyAnnotationsToField(field, annotations)
		return parseRecordFieldName(field), nil
	}
}

func parseRecordFieldName(field *Field) parserAction[*Record] {
	return parseIdent(func(tok Token) (parserAction[*Record], error) {
		field.Name = string(tok.Value)
		return parseRecordFieldDefaultOrSemicolon(field), nil
	})
}

func parseRecordFieldDefaultOrSemicolon(field *Field) parserAction[*Record] {
	return func(p *parser, rec *Record) (parserAction[*Record], error) {
		tok, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		switch {
		case bytes.Equal(tok.Value, []byte("=")):
			return parseRecordFieldDefaultValue(field), nil
		case bytes.Equal(tok.Value, []byte(";")):
			return parseRecordFieldTypeOrClose, nil
		default:
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   tok,
			}
		}
	}
}

func parseRecordFieldDefaultValue(field *Field) parserAction[*Record] {
	return func(p *parser, rec *Record) (parserAction[*Record], error) {
		val, err := parseJSONValue(p)
		if err != nil {
			return nil, err
		}
		field.Default = val
		return parseRecordFieldSemicolon, nil
	}
}

func parseJSONValue(p *parser) (Value, error) {
	tok, err := p.expect(TokenIdentifier, TokenNumber, TokenString, TokenSymbol)
	if err != nil {
		return nil, err
	}
	switch tok.Type {
	case TokenIdentifier:
		switch string(tok.Value) {
		case "null":
			return NullValue{}, nil
		case "true":
			return BoolValue(true), nil
		case "false":
			return BoolValue(false), nil
		default:
			return StringValue(tok.Value), nil
		}
	case TokenNumber:
		s := string(tok.Value)
		if strings.Contains(s, ".") {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, err
			}
			return FloatValue(f), nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return IntValue(i), nil
	case TokenString:
		return StringValue(tok.Value), nil
	case TokenSymbol:
		switch {
		case bytes.Equal(tok.Value, []byte("[")):
			return parseJSONArray(p)
		case bytes.Equal(tok.Value, []byte("{")):
			return parseJSONObject(p)
		default:
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   tok,
			}
		}
	default:
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenNumber, TokenString, TokenSymbol},
			Actual:   tok,
		}
	}
}

func parseJSONArray(p *parser) (ArrayValue, error) {
	var arr ArrayValue
	tok, err := p.expect(TokenIdentifier, TokenNumber, TokenString, TokenSymbol)
	if err != nil {
		return nil, err
	}
	if tok.Type == TokenSymbol && bytes.Equal(tok.Value, []byte("]")) {
		return arr, nil
	}
	p.unread(tok)
	for {
		val, err := parseJSONValue(p)
		if err != nil {
			return nil, err
		}
		arr = append(arr, val)
		sep, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(sep.Value, []byte("]")) {
			return arr, nil
		}
		if !bytes.Equal(sep.Value, []byte(",")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   sep,
			}
		}
	}
}

func parseJSONObject(p *parser) (ObjectValue, error) {
	obj := make(ObjectValue)
	tok, err := p.expect(TokenString, TokenSymbol)
	if err != nil {
		return nil, err
	}
	if tok.Type == TokenSymbol && bytes.Equal(tok.Value, []byte("}")) {
		return obj, nil
	}
	if tok.Type != TokenString {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenString},
			Actual:   tok,
		}
	}
	for {
		key := string(tok.Value)
		colon, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(colon.Value, []byte(":")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   colon,
			}
		}
		val, err := parseJSONValue(p)
		if err != nil {
			return nil, err
		}
		obj[key] = val
		sep, err := p.expect(TokenSymbol)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(sep.Value, []byte("}")) {
			return obj, nil
		}
		if !bytes.Equal(sep.Value, []byte(",")) {
			return nil, UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   sep,
			}
		}
		tok, err = p.expect(TokenString)
		if err != nil {
			return nil, err
		}
	}
}

func parseRecordFieldSemicolon(p *parser, rec *Record) (parserAction[*Record], error) {
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
	return parseRecordFieldTypeOrClose, nil
}

func parseRecordFieldTypeOrClose(p *parser, rec *Record) (parserAction[*Record], error) {
	doc, err := collectDocComment(p)
	if err != nil {
		return nil, err
	}
	preTypeAnnotations, err := collectAnnotations(p)
	if err != nil {
		return nil, err
	}

	tok, err, ok := p.read()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenSymbol}}
	}

	// Check for closing brace
	if tok.Type == TokenSymbol {
		if bytes.Equal(tok.Value, []byte("}")) {
			return nil, nil
		}
		// Check for escaped identifier starting with backtick
		if bytes.Equal(tok.Value, []byte("`")) {
			p.unread(tok)
			identTok, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			field := &Field{Doc: doc, Type: Ident{Pos: identTok.Pos, Value: string(identTok.Value)}}
			applyAnnotationsToField(field, preTypeAnnotations)
			rec.Fields = append(rec.Fields, field)
			return parseRecordFieldNullableOrName(field), nil
		}
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}

	if tok.Type != TokenIdentifier {
		return nil, UnexpectedTokenError{
			Expected: []TokenType{TokenIdentifier, TokenSymbol},
			Actual:   tok,
		}
	}

	var typ Type
	switch string(tok.Value) {
	case "map":
		m, err := parseMapType(p)
		if err != nil {
			return nil, err
		}
		typ = m
	case "union":
		u, err := parseUnionType(p)
		if err != nil {
			return nil, err
		}
		typ = u
	default:
		typ = Ident{Pos: tok.Pos, Value: string(tok.Value)}
	}
	field := &Field{Doc: doc, Type: typ}
	applyAnnotationsToField(field, preTypeAnnotations)
	rec.Fields = append(rec.Fields, field)
	return parseRecordFieldNullableOrName(field), nil
}
