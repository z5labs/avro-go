// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Primitive represents a primitive Avro type in canonical form.
type Primitive string

const (
	Null    Primitive = "null"
	Boolean Primitive = "boolean"
	Int     Primitive = "int"
	Long    Primitive = "long"
	Float   Primitive = "float"
	Double  Primitive = "double"
	Bytes   Primitive = "bytes"
	String  Primitive = "string"
)

// Record represents a record schema in canonical form.
type Record struct {
	Name   string
	Fields []Field
}

// Field represents a field within a record schema in canonical form.
type Field struct {
	Name string
	Type Schema
}

// Enum represents an enum schema in canonical form.
type Enum struct {
	Name    string
	Symbols []string
}

// Array represents an array schema in canonical form.
type Array struct {
	Items Schema
}

// Map represents a map schema in canonical form.
type Map struct {
	Values Schema
}

// Union represents a union schema in canonical form.
type Union []Schema

// Fixed represents a fixed schema in canonical form.
type Fixed struct {
	Name string
	Size int
}

// Schema is the top-level type that represents any Avro schema in parsing canonical form.
// It implements json.Marshaler and json.Unmarshaler.
type Schema struct {
	val any
}

// PrimitiveSchema creates a Schema from a Primitive.
func PrimitiveSchema(p Primitive) Schema { return Schema{val: p} }

// RecordSchema creates a Schema from a Record.
func RecordSchema(r Record) Schema { return Schema{val: r} }

// EnumSchema creates a Schema from an Enum.
func EnumSchema(e Enum) Schema { return Schema{val: e} }

// ArraySchema creates a Schema from an Array.
func ArraySchema(a Array) Schema { return Schema{val: a} }

// MapSchema creates a Schema from a Map.
func MapSchema(m Map) Schema { return Schema{val: m} }

// UnionSchema creates a Schema from a Union.
func UnionSchema(u Union) Schema { return Schema{val: u} }

// FixedSchema creates a Schema from a Fixed.
func FixedSchema(f Fixed) Schema { return Schema{val: f} }

// Primitive returns the underlying Primitive value and true if the Schema holds a Primitive.
func (s Schema) Primitive() (Primitive, bool) {
	p, ok := s.val.(Primitive)
	return p, ok
}

// Record returns the underlying Record value and true if the Schema holds a Record.
func (s Schema) Record() (Record, bool) {
	r, ok := s.val.(Record)
	return r, ok
}

// Enum returns the underlying Enum value and true if the Schema holds an Enum.
func (s Schema) Enum() (Enum, bool) {
	e, ok := s.val.(Enum)
	return e, ok
}

// Array returns the underlying Array value and true if the Schema holds an Array.
func (s Schema) Array() (Array, bool) {
	a, ok := s.val.(Array)
	return a, ok
}

// Map returns the underlying Map value and true if the Schema holds a Map.
func (s Schema) Map() (Map, bool) {
	m, ok := s.val.(Map)
	return m, ok
}

// Union returns the underlying Union value and true if the Schema holds a Union.
func (s Schema) Union() (Union, bool) {
	u, ok := s.val.(Union)
	return u, ok
}

// Fixed returns the underlying Fixed value and true if the Schema holds a Fixed.
func (s Schema) Fixed() (Fixed, bool) {
	f, ok := s.val.(Fixed)
	return f, ok
}

// MarshalJSON implements json.Marshaler. It produces canonical form JSON
// with correct field ordering (name, type, fields, symbols, items, values, size)
// and no extra whitespace.
func (s Schema) MarshalJSON() ([]byte, error) {
	switch v := s.val.(type) {
	case Primitive:
		return marshalPrimitive(v), nil
	case Record:
		return marshalRecord(v)
	case Enum:
		return marshalEnum(v)
	case Array:
		return marshalArray(v)
	case Map:
		return marshalMap(v)
	case Union:
		return marshalUnion(v)
	case Fixed:
		return marshalFixed(v), nil
	default:
		return nil, fmt.Errorf("canonical: unknown schema type %T", v)
	}
}

func marshalPrimitive(p Primitive) []byte {
	b := make([]byte, 0, len(p)+2)
	b = append(b, '"')
	b = append(b, p...)
	b = append(b, '"')
	return b
}

func marshalRecord(r Record) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"name":`)
	writeQuotedString(&buf, r.Name)
	buf.WriteString(`,"type":"record","fields":[`)
	for i, f := range r.Fields {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := marshalField(&buf, f); err != nil {
			return nil, err
		}
	}
	buf.WriteString(`]}`)
	return buf.Bytes(), nil
}

func marshalField(buf *bytes.Buffer, f Field) error {
	buf.WriteString(`{"name":`)
	writeQuotedString(buf, f.Name)
	buf.WriteString(`,"type":`)
	b, err := f.Type.MarshalJSON()
	if err != nil {
		return err
	}
	buf.Write(b)
	buf.WriteByte('}')
	return nil
}

func marshalEnum(e Enum) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"name":`)
	writeQuotedString(&buf, e.Name)
	buf.WriteString(`,"type":"enum","symbols":[`)
	for i, s := range e.Symbols {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeQuotedString(&buf, s)
	}
	buf.WriteString(`]}`)
	return buf.Bytes(), nil
}

func marshalArray(a Array) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"type":"array","items":`)
	b, err := a.Items.MarshalJSON()
	if err != nil {
		return nil, err
	}
	buf.Write(b)
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalMap(m Map) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"type":"map","values":`)
	b, err := m.Values.MarshalJSON()
	if err != nil {
		return nil, err
	}
	buf.Write(b)
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalUnion(u Union) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, s := range u {
		if i > 0 {
			buf.WriteByte(',')
		}
		b, err := s.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func marshalFixed(f Fixed) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"name":`)
	writeQuotedString(&buf, f.Name)
	buf.WriteString(`,"type":"fixed","size":`)
	buf.WriteString(strconv.Itoa(f.Size))
	buf.WriteByte('}')
	return buf.Bytes()
}

func writeQuotedString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if c < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, c)
			} else {
				buf.WriteRune(c)
			}
		}
	}
	buf.WriteByte('"')
}

// UnmarshalJSON implements json.Unmarshaler. It parses canonical form JSON
// into the appropriate concrete schema type.
func (s *Schema) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("canonical: empty JSON")
	}

	switch data[0] {
	case '"':
		var p string
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		s.val = Primitive(p)
		return nil
	case '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		u := make(Union, len(raw))
		for i, r := range raw {
			if err := u[i].UnmarshalJSON(r); err != nil {
				return err
			}
		}
		s.val = u
		return nil
	case '{':
		return s.unmarshalObject(data)
	default:
		return fmt.Errorf("canonical: unexpected JSON token %q", data[0])
	}
}

func (s *Schema) unmarshalObject(data []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	rawType, ok := obj["type"]
	if !ok {
		return fmt.Errorf("canonical: object schema missing \"type\" field")
	}

	var typ string
	if err := json.Unmarshal(rawType, &typ); err != nil {
		return err
	}

	switch typ {
	case "record":
		return s.unmarshalRecord(obj)
	case "enum":
		return s.unmarshalEnum(obj)
	case "array":
		return s.unmarshalArray(obj)
	case "map":
		return s.unmarshalMap(obj)
	case "fixed":
		return s.unmarshalFixed(obj)
	default:
		return fmt.Errorf("canonical: unknown type %q", typ)
	}
}

func (s *Schema) unmarshalRecord(obj map[string]json.RawMessage) error {
	rawName, ok := obj["name"]
	if !ok {
		return fmt.Errorf("canonical: record: missing name")
	}
	var name string
	if err := json.Unmarshal(rawName, &name); err != nil {
		return fmt.Errorf("canonical: record: invalid name: %w", err)
	}

	rawFields, ok := obj["fields"]
	if !ok {
		return fmt.Errorf("canonical: record: missing fields")
	}
	var rawFieldList []json.RawMessage
	if err := json.Unmarshal(rawFields, &rawFieldList); err != nil {
		return fmt.Errorf("canonical: record: invalid fields: %w", err)
	}

	fields := make([]Field, len(rawFieldList))
	for i, rf := range rawFieldList {
		var fieldObj map[string]json.RawMessage
		if err := json.Unmarshal(rf, &fieldObj); err != nil {
			return fmt.Errorf("canonical: record field: %w", err)
		}

		nameRaw, ok := fieldObj["name"]
		if !ok {
			return fmt.Errorf("canonical: record field: missing name")
		}
		var fieldName string
		if err := json.Unmarshal(nameRaw, &fieldName); err != nil {
			return fmt.Errorf("canonical: record field: invalid name: %w", err)
		}

		typeRaw, ok := fieldObj["type"]
		if !ok {
			return fmt.Errorf("canonical: record field: missing type")
		}
		var fieldType Schema
		if err := fieldType.UnmarshalJSON(typeRaw); err != nil {
			return fmt.Errorf("canonical: record field: invalid type: %w", err)
		}

		fields[i] = Field{Name: fieldName, Type: fieldType}
	}

	s.val = Record{Name: name, Fields: fields}
	return nil
}

func (s *Schema) unmarshalEnum(obj map[string]json.RawMessage) error {
	rawName, ok := obj["name"]
	if !ok {
		return fmt.Errorf("canonical: enum: missing name")
	}
	var name string
	if err := json.Unmarshal(rawName, &name); err != nil {
		return fmt.Errorf("canonical: enum: invalid name: %w", err)
	}

	rawSymbols, ok := obj["symbols"]
	if !ok {
		return fmt.Errorf("canonical: enum: missing symbols")
	}
	var symbols []string
	if err := json.Unmarshal(rawSymbols, &symbols); err != nil {
		return fmt.Errorf("canonical: enum: invalid symbols: %w", err)
	}

	s.val = Enum{Name: name, Symbols: symbols}
	return nil
}

func (s *Schema) unmarshalArray(obj map[string]json.RawMessage) error {
	rawItems, ok := obj["items"]
	if !ok {
		return fmt.Errorf("canonical: array: missing items")
	}
	var items Schema
	if err := items.UnmarshalJSON(rawItems); err != nil {
		return fmt.Errorf("canonical: array: invalid items: %w", err)
	}

	s.val = Array{Items: items}
	return nil
}

func (s *Schema) unmarshalMap(obj map[string]json.RawMessage) error {
	rawValues, ok := obj["values"]
	if !ok {
		return fmt.Errorf("canonical: map: missing values")
	}
	var values Schema
	if err := values.UnmarshalJSON(rawValues); err != nil {
		return fmt.Errorf("canonical: map: invalid values: %w", err)
	}

	s.val = Map{Values: values}
	return nil
}

func (s *Schema) unmarshalFixed(obj map[string]json.RawMessage) error {
	rawName, ok := obj["name"]
	if !ok {
		return fmt.Errorf("canonical: fixed: missing name")
	}
	var name string
	if err := json.Unmarshal(rawName, &name); err != nil {
		return fmt.Errorf("canonical: fixed: invalid name: %w", err)
	}

	rawSize, ok := obj["size"]
	if !ok {
		return fmt.Errorf("canonical: fixed: missing size")
	}
	var size int
	if err := json.Unmarshal(rawSize, &size); err != nil {
		return fmt.Errorf("canonical: fixed: invalid size: %w", err)
	}

	s.val = Fixed{Name: name, Size: size}
	return nil
}
