// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import "io"

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
	Namespace string
	Type      Type
}

// Protocol represents a protocol in the Avro IDL.
type Protocol struct {
	Namespace string
}

// File represents a parsed Avro IDL file. Each Avro IDL file defines either a
// single Avro Protocol, or an Avro Schema with supporting named schemata in a
// namespace.
type File struct {
	Schema   *Schema
	Protocol *Protocol
}

// Parse the Avro IDL defined in the given reader.
func Parse(r io.Reader) (*File, error) {
	return &File{}, nil
}
