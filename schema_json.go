// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ParseJSON parses an Avro schema in its JSON definition form
// (https://avro.apache.org/docs/1.12.0/specification/) into a Schema.
//
// All concrete Schema types in this package implement json.Marshaler so a
// Schema can be written back to JSON via json.Marshal.
func ParseJSON(data []byte) (Schema, error) {
	return parseSchemaJSON(data, "")
}

// MissingFieldError reports a JSON object schema that lacks a required field
// (e.g. a record schema with no "fields" entry).
type MissingFieldError struct {
	Type  string // schema kind: "record", "enum", "array", etc.
	Field string // missing JSON field name
}

func (e MissingFieldError) Error() string {
	return fmt.Sprintf("avro: %s schema: missing %q field", e.Type, e.Field)
}

// UnknownTypeError reports a JSON schema whose declared "type" is not a known
// primitive, complex type, or named-type reference within scope.
type UnknownTypeError struct{ Type string }

func (e UnknownTypeError) Error() string {
	return fmt.Sprintf("avro: unknown schema type %q", e.Type)
}

// UnknownLogicalTypeError reports a JSON schema with a "logicalType" the
// parser does not recognize.
type UnknownLogicalTypeError struct {
	Underlying  string
	LogicalType string
}

func (e UnknownLogicalTypeError) Error() string {
	return fmt.Sprintf("avro: unknown logicalType %q on %q", e.LogicalType, e.Underlying)
}

// ErrEmptyJSON is returned when ParseJSON is given empty input.
var ErrEmptyJSON = errors.New("avro: empty JSON")

// UnexpectedTokenError reports a JSON document that starts with a token
// other than '"', '[', or '{'.
type UnexpectedTokenError struct{ Token byte }

func (e UnexpectedTokenError) Error() string {
	return fmt.Sprintf("avro: unexpected JSON token %q", e.Token)
}

// InvalidDecimalUnderlyingError reports a Decimal whose Underlying schema is
// not Bytes or Fixed when marshaling to JSON.
type InvalidDecimalUnderlyingError struct{ Underlying Schema }

func (e InvalidDecimalUnderlyingError) Error() string {
	return fmt.Sprintf("avro: decimal underlying must be bytes or fixed, got %T", e.Underlying)
}

// ---- primitive marshalers ----

func (Null) MarshalJSON() ([]byte, error)    { return []byte(`"null"`), nil }
func (Boolean) MarshalJSON() ([]byte, error) { return []byte(`"boolean"`), nil }
func (Int) MarshalJSON() ([]byte, error)     { return []byte(`"int"`), nil }
func (Long) MarshalJSON() ([]byte, error)    { return []byte(`"long"`), nil }
func (Float) MarshalJSON() ([]byte, error)   { return []byte(`"float"`), nil }
func (Double) MarshalJSON() ([]byte, error)  { return []byte(`"double"`), nil }
func (Bytes) MarshalJSON() ([]byte, error)   { return []byte(`"bytes"`), nil }
func (String) MarshalJSON() ([]byte, error)  { return []byte(`"string"`), nil }

func (r Ref) MarshalJSON() ([]byte, error) {
	return json.Marshal(qualifiedName(r.Namespace, r.Name))
}

// ---- composite marshalers ----

func (r Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      string   `json:"type"`
		Name      string   `json:"name"`
		Namespace string   `json:"namespace,omitempty"`
		Doc       string   `json:"doc,omitempty"`
		Aliases   []string `json:"aliases,omitempty"`
		Fields    []*Field `json:"fields"`
	}{"record", r.Name, r.Namespace, r.Doc, r.Aliases, r.Fields})
}

func (f Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name    string   `json:"name"`
		Doc     string   `json:"doc,omitempty"`
		Aliases []string `json:"aliases,omitempty"`
		Type    Schema   `json:"type"`
	}{f.Name, f.Doc, f.Aliases, f.Type})
}

func (e Enum) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      string   `json:"type"`
		Name      string   `json:"name"`
		Namespace string   `json:"namespace,omitempty"`
		Doc       string   `json:"doc,omitempty"`
		Aliases   []string `json:"aliases,omitempty"`
		Symbols   []string `json:"symbols"`
		Default   string   `json:"default,omitempty"`
	}{"enum", e.Name, e.Namespace, e.Doc, e.Aliases, e.Symbols, e.Default})
}

func (a Array) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string `json:"type"`
		Items Schema `json:"items"`
	}{"array", a.Items})
}

func (m Map) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type   string `json:"type"`
		Values Schema `json:"values"`
	}{"map", m.Values})
}

func (u Union) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Types)
}

func (f Fixed) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      string   `json:"type"`
		Name      string   `json:"name"`
		Namespace string   `json:"namespace,omitempty"`
		Aliases   []string `json:"aliases,omitempty"`
		Size      int      `json:"size"`
	}{"fixed", f.Name, f.Namespace, f.Aliases, f.Size})
}

// ---- logical-type marshalers ----

func (d Decimal) MarshalJSON() ([]byte, error) {
	switch under := d.Underlying.(type) {
	case Bytes:
		return json.Marshal(struct {
			Type        string `json:"type"`
			LogicalType string `json:"logicalType"`
			Precision   int    `json:"precision"`
			Scale       int    `json:"scale,omitempty"`
		}{"bytes", "decimal", d.Precision, d.Scale})
	case Fixed:
		return json.Marshal(struct {
			Type        string   `json:"type"`
			LogicalType string   `json:"logicalType"`
			Name        string   `json:"name"`
			Namespace   string   `json:"namespace,omitempty"`
			Aliases     []string `json:"aliases,omitempty"`
			Size        int      `json:"size"`
			Precision   int      `json:"precision"`
			Scale       int      `json:"scale,omitempty"`
		}{"fixed", "decimal", under.Name, under.Namespace, under.Aliases, under.Size, d.Precision, d.Scale})
	default:
		return nil, InvalidDecimalUnderlyingError{Underlying: d.Underlying}
	}
}

func (UUID) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"string","logicalType":"uuid"}`), nil
}

func (Date) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"int","logicalType":"date"}`), nil
}

func (TimeMillis) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"int","logicalType":"time-millis"}`), nil
}

func (TimeMicros) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"time-micros"}`), nil
}

func (TimestampMillis) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"timestamp-millis"}`), nil
}

func (TimestampMicros) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"timestamp-micros"}`), nil
}

func (TimestampNanos) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"timestamp-nanos"}`), nil
}

func (LocalTimestampMillis) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"local-timestamp-millis"}`), nil
}

func (LocalTimestampMicros) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"local-timestamp-micros"}`), nil
}

func (LocalTimestampNanos) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"long","logicalType":"local-timestamp-nanos"}`), nil
}

func (Duration) MarshalJSON() ([]byte, error) {
	// Avro requires fixed types to have a name; "duration" is conventional.
	return []byte(`{"type":"fixed","name":"duration","size":12,"logicalType":"duration"}`), nil
}

// ---- parser ----

func parseSchemaJSON(data []byte, namespace string) (Schema, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, ErrEmptyJSON
	}
	switch data[0] {
	case '"':
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return nil, err
		}
		return parseTypeName(name, namespace), nil
	case '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		types := make([]Schema, len(raw))
		for i, r := range raw {
			t, err := parseSchemaJSON(r, namespace)
			if err != nil {
				return nil, err
			}
			types[i] = t
		}
		return Union{Types: types}, nil
	case '{':
		return parseObjectJSON(data, namespace)
	default:
		return nil, UnexpectedTokenError{Token: data[0]}
	}
}

func parseTypeName(name, namespace string) Schema {
	switch name {
	case "null":
		return Null{}
	case "boolean":
		return Boolean{}
	case "int":
		return Int{}
	case "long":
		return Long{}
	case "float":
		return Float{}
	case "double":
		return Double{}
	case "bytes":
		return Bytes{}
	case "string":
		return String{}
	}
	// Otherwise it's a named-type reference. If the name is qualified, split it;
	// otherwise inherit the enclosing namespace.
	if i := strings.LastIndex(name, "."); i >= 0 {
		return Ref{Namespace: name[:i], Name: name[i+1:]}
	}
	return Ref{Name: name, Namespace: namespace}
}

func parseObjectJSON(data []byte, namespace string) (Schema, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	typeRaw, ok := obj["type"]
	if !ok {
		return nil, MissingFieldError{Type: "object", Field: "type"}
	}
	var typeName string
	if err := json.Unmarshal(typeRaw, &typeName); err != nil {
		// "type" is itself a nested schema object/array.
		return parseSchemaJSON(typeRaw, namespace)
	}

	// Logical-type schemas dispatch on the logicalType field.
	if ltRaw, hasLogical := obj["logicalType"]; hasLogical {
		var lt string
		if err := json.Unmarshal(ltRaw, &lt); err != nil {
			return nil, err
		}
		return parseLogical(typeName, lt, obj, namespace)
	}

	switch typeName {
	case "null", "boolean", "int", "long", "float", "double", "bytes", "string":
		return parseTypeName(typeName, namespace), nil
	case "record":
		return parseRecord(obj, namespace)
	case "enum":
		return parseEnum(obj, namespace)
	case "array":
		return parseArray(obj, namespace)
	case "map":
		return parseMap(obj, namespace)
	case "fixed":
		return parseFixed(obj, namespace)
	default:
		// Could be a named-type reference (rarely written in object form).
		ref := parseTypeName(typeName, namespace)
		if _, isRef := ref.(Ref); isRef {
			return ref, nil
		}
		return nil, UnknownTypeError{Type: typeName}
	}
}

func parseRecord(obj map[string]json.RawMessage, namespace string) (Record, error) {
	r := Record{}
	if err := unmarshalIfPresent(obj, "name", &r.Name); err != nil {
		return r, err
	}
	if r.Name == "" {
		return r, MissingFieldError{Type: "record", Field: "name"}
	}
	if err := unmarshalIfPresent(obj, "namespace", &r.Namespace); err != nil {
		return r, err
	}
	if err := unmarshalIfPresent(obj, "doc", &r.Doc); err != nil {
		return r, err
	}
	if err := unmarshalIfPresent(obj, "aliases", &r.Aliases); err != nil {
		return r, err
	}
	ns := r.Namespace
	if ns == "" {
		ns = namespace
	}

	rawFields, ok := obj["fields"]
	if !ok {
		return r, MissingFieldError{Type: "record", Field: "fields"}
	}
	var rawList []json.RawMessage
	if err := json.Unmarshal(rawFields, &rawList); err != nil {
		return r, err
	}
	r.Fields = make([]*Field, len(rawList))
	for i, raw := range rawList {
		f, err := parseField(raw, ns)
		if err != nil {
			return r, err
		}
		r.Fields[i] = f
	}
	return r, nil
}

func parseField(data []byte, namespace string) (*Field, error) {
	var fobj map[string]json.RawMessage
	if err := json.Unmarshal(data, &fobj); err != nil {
		return nil, err
	}
	f := &Field{}
	if err := unmarshalIfPresent(fobj, "name", &f.Name); err != nil {
		return nil, err
	}
	if f.Name == "" {
		return nil, MissingFieldError{Type: "field", Field: "name"}
	}
	if err := unmarshalIfPresent(fobj, "doc", &f.Doc); err != nil {
		return nil, err
	}
	if err := unmarshalIfPresent(fobj, "aliases", &f.Aliases); err != nil {
		return nil, err
	}
	rawType, ok := fobj["type"]
	if !ok {
		return nil, MissingFieldError{Type: "field", Field: "type"}
	}
	t, err := parseSchemaJSON(rawType, namespace)
	if err != nil {
		return nil, err
	}
	f.Type = t
	return f, nil
}

func parseEnum(obj map[string]json.RawMessage, namespace string) (Enum, error) {
	_ = namespace
	e := Enum{}
	if err := unmarshalIfPresent(obj, "name", &e.Name); err != nil {
		return e, err
	}
	if e.Name == "" {
		return e, MissingFieldError{Type: "enum", Field: "name"}
	}
	if err := unmarshalIfPresent(obj, "namespace", &e.Namespace); err != nil {
		return e, err
	}
	if err := unmarshalIfPresent(obj, "doc", &e.Doc); err != nil {
		return e, err
	}
	if err := unmarshalIfPresent(obj, "aliases", &e.Aliases); err != nil {
		return e, err
	}
	if err := unmarshalIfPresent(obj, "symbols", &e.Symbols); err != nil {
		return e, err
	}
	if e.Symbols == nil {
		return e, MissingFieldError{Type: "enum", Field: "symbols"}
	}
	if err := unmarshalIfPresent(obj, "default", &e.Default); err != nil {
		return e, err
	}
	return e, nil
}

func parseArray(obj map[string]json.RawMessage, namespace string) (Array, error) {
	rawItems, ok := obj["items"]
	if !ok {
		return Array{}, MissingFieldError{Type: "array", Field: "items"}
	}
	items, err := parseSchemaJSON(rawItems, namespace)
	if err != nil {
		return Array{}, err
	}
	return Array{Items: items}, nil
}

func parseMap(obj map[string]json.RawMessage, namespace string) (Map, error) {
	rawValues, ok := obj["values"]
	if !ok {
		return Map{}, MissingFieldError{Type: "map", Field: "values"}
	}
	values, err := parseSchemaJSON(rawValues, namespace)
	if err != nil {
		return Map{}, err
	}
	return Map{Values: values}, nil
}

func parseFixed(obj map[string]json.RawMessage, _ string) (Fixed, error) {
	f := Fixed{}
	if err := unmarshalIfPresent(obj, "name", &f.Name); err != nil {
		return f, err
	}
	if f.Name == "" {
		return f, MissingFieldError{Type: "fixed", Field: "name"}
	}
	if err := unmarshalIfPresent(obj, "namespace", &f.Namespace); err != nil {
		return f, err
	}
	if err := unmarshalIfPresent(obj, "aliases", &f.Aliases); err != nil {
		return f, err
	}
	if _, ok := obj["size"]; !ok {
		return f, MissingFieldError{Type: "fixed", Field: "size"}
	}
	if err := unmarshalIfPresent(obj, "size", &f.Size); err != nil {
		return f, err
	}
	return f, nil
}

func parseLogical(under, lt string, obj map[string]json.RawMessage, namespace string) (Schema, error) {
	switch lt {
	case "decimal":
		var precision, scale int
		if err := unmarshalIfPresent(obj, "precision", &precision); err != nil {
			return nil, err
		}
		if err := unmarshalIfPresent(obj, "scale", &scale); err != nil {
			return nil, err
		}
		switch under {
		case "bytes":
			return Decimal{Precision: precision, Scale: scale, Underlying: Bytes{}}, nil
		case "fixed":
			fx, err := parseFixed(obj, namespace)
			if err != nil {
				return nil, err
			}
			return Decimal{Precision: precision, Scale: scale, Underlying: fx}, nil
		default:
			return nil, UnknownLogicalTypeError{Underlying: under, LogicalType: lt}
		}
	case "uuid":
		return UUID{}, nil
	case "date":
		return Date{}, nil
	case "time-millis":
		return TimeMillis{}, nil
	case "time-micros":
		return TimeMicros{}, nil
	case "timestamp-millis":
		return TimestampMillis{}, nil
	case "timestamp-micros":
		return TimestampMicros{}, nil
	case "timestamp-nanos":
		return TimestampNanos{}, nil
	case "local-timestamp-millis":
		return LocalTimestampMillis{}, nil
	case "local-timestamp-micros":
		return LocalTimestampMicros{}, nil
	case "local-timestamp-nanos":
		return LocalTimestampNanos{}, nil
	case "duration":
		return Duration{}, nil
	default:
		return nil, UnknownLogicalTypeError{Underlying: under, LogicalType: lt}
	}
}

// ---- helpers ----

func unmarshalIfPresent(obj map[string]json.RawMessage, key string, dst any) error {
	raw, ok := obj[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, dst)
}

func qualifiedName(namespace, name string) string {
	if name == "" || strings.Contains(name, ".") || namespace == "" {
		return name
	}
	return namespace + "." + name
}
