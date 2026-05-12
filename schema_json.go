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

// InvalidDurationSizeError reports a Duration whose Underlying.Size is set
// to a value other than 12 (the size required by the Avro spec).
type InvalidDurationSizeError struct{ Size int }

func (e InvalidDurationSizeError) Error() string {
	return fmt.Sprintf("avro: duration underlying size must be 12, got %d", e.Size)
}

// InvalidOrderError reports a Field with an Order outside the spec's
// {ascending, descending, ignore} set.
type InvalidOrderError struct{ Order Order }

func (e InvalidOrderError) Error() string {
	return fmt.Sprintf("avro: invalid field order %d", int(e.Order))
}

// InvalidOrderStringError reports a Field whose JSON "order" value is not
// one of "ascending", "descending", or "ignore".
type InvalidOrderStringError struct{ Value string }

func (e InvalidOrderStringError) Error() string {
	return fmt.Sprintf("avro: invalid field order %q", e.Value)
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
	var order string
	switch f.Order {
	case OrderAscending:
		// spec default; omit
	case OrderDescending:
		order = "descending"
	case OrderIgnore:
		order = "ignore"
	default:
		return nil, InvalidOrderError{Order: f.Order}
	}
	// json.RawMessage with omitempty lets us distinguish "no default" (nil
	// slice, omitted) from an explicit null default (the bytes "null", emitted).
	var def json.RawMessage
	if f.HasDefault {
		b, err := json.Marshal(f.Default)
		if err != nil {
			return nil, err
		}
		def = b
	}
	return json.Marshal(struct {
		Name    string          `json:"name"`
		Doc     string          `json:"doc,omitempty"`
		Aliases []string        `json:"aliases,omitempty"`
		Type    Schema          `json:"type"`
		Default json.RawMessage `json:"default,omitempty"`
		Order   string          `json:"order,omitempty"`
	}{f.Name, f.Doc, f.Aliases, f.Type, def, order})
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

func (d Duration) MarshalJSON() ([]byte, error) {
	// The Avro spec fixes a duration's underlying fixed size at 12 bytes. Size
	// 0 means the caller did not specify a size (we then emit the canonical
	// 12); any other value would silently misrepresent the in-memory schema.
	if d.Underlying.Size != 0 && d.Underlying.Size != 12 {
		return nil, InvalidDurationSizeError{Size: d.Underlying.Size}
	}
	// "duration" is the conventional name when none is supplied; the Underlying
	// Fixed otherwise carries the user-chosen name/namespace/aliases so that
	// JSON round-trips preserve identity.
	name := d.Underlying.Name
	if name == "" {
		name = "duration"
	}
	return json.Marshal(struct {
		Type        string   `json:"type"`
		LogicalType string   `json:"logicalType"`
		Name        string   `json:"name"`
		Namespace   string   `json:"namespace,omitempty"`
		Aliases     []string `json:"aliases,omitempty"`
		Size        int      `json:"size"`
	}{"fixed", "duration", name, d.Underlying.Namespace, d.Underlying.Aliases, 12})
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
	if rawDefault, ok := fobj["default"]; ok {
		f.HasDefault = true
		var v any
		if err := json.Unmarshal(rawDefault, &v); err != nil {
			return nil, err
		}
		f.Default = v
	}
	if rawOrder, ok := fobj["order"]; ok {
		var s string
		if err := json.Unmarshal(rawOrder, &s); err != nil {
			return nil, err
		}
		switch s {
		case "ascending":
			f.Order = OrderAscending
		case "descending":
			f.Order = OrderDescending
		case "ignore":
			f.Order = OrderIgnore
		default:
			return nil, InvalidOrderStringError{Value: s}
		}
	}
	return f, nil
}

func parseEnum(obj map[string]json.RawMessage, namespace string) (Enum, error) {
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
	if e.Namespace == "" {
		e.Namespace = namespace
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

func parseFixed(obj map[string]json.RawMessage, namespace string) (Fixed, error) {
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
	if f.Namespace == "" {
		f.Namespace = namespace
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

// parseLogical decodes an Avro JSON object whose declared "logicalType" field
// is non-empty. Per the Avro spec, when the underlying type does not match the
// logical type's required base (or the logical type is unknown), the logical
// type is ignored and the underlying type is returned as the schema.
func parseLogical(under, lt string, obj map[string]json.RawMessage, namespace string) (Schema, error) {
	underlying, err := parseLogicalUnderlying(under, obj, namespace)
	if err != nil {
		return nil, err
	}

	switch lt {
	case "decimal":
		if under != "bytes" && under != "fixed" {
			return underlying, nil
		}
		if _, ok := obj["precision"]; !ok {
			return nil, MissingFieldError{Type: "decimal", Field: "precision"}
		}
		var precision, scale int
		if err := unmarshalIfPresent(obj, "precision", &precision); err != nil {
			return nil, err
		}
		if err := unmarshalIfPresent(obj, "scale", &scale); err != nil {
			return nil, err
		}
		return Decimal{Precision: precision, Scale: scale, Underlying: underlying}, nil
	case "uuid":
		if under != "string" {
			return underlying, nil
		}
		return UUID{}, nil
	case "date":
		if under != "int" {
			return underlying, nil
		}
		return Date{}, nil
	case "time-millis":
		if under != "int" {
			return underlying, nil
		}
		return TimeMillis{}, nil
	case "time-micros":
		if under != "long" {
			return underlying, nil
		}
		return TimeMicros{}, nil
	case "timestamp-millis":
		if under != "long" {
			return underlying, nil
		}
		return TimestampMillis{}, nil
	case "timestamp-micros":
		if under != "long" {
			return underlying, nil
		}
		return TimestampMicros{}, nil
	case "timestamp-nanos":
		if under != "long" {
			return underlying, nil
		}
		return TimestampNanos{}, nil
	case "local-timestamp-millis":
		if under != "long" {
			return underlying, nil
		}
		return LocalTimestampMillis{}, nil
	case "local-timestamp-micros":
		if under != "long" {
			return underlying, nil
		}
		return LocalTimestampMicros{}, nil
	case "local-timestamp-nanos":
		if under != "long" {
			return underlying, nil
		}
		return LocalTimestampNanos{}, nil
	case "duration":
		fx, ok := underlying.(Fixed)
		if !ok || fx.Size != 12 {
			return underlying, nil
		}
		return Duration{Underlying: fx}, nil
	default:
		// Unknown logical type: per spec, ignore and use the underlying type.
		return underlying, nil
	}
}

// parseLogicalUnderlying resolves the "type" field of a logical-type schema
// into its base Schema. Per the Avro spec's fallback rule, the underlying
// schema must be parseable for any valid Avro type so that an invalid or
// unknown logical type can be ignored cleanly.
func parseLogicalUnderlying(under string, obj map[string]json.RawMessage, namespace string) (Schema, error) {
	switch under {
	case "null", "boolean", "int", "long", "float", "double", "bytes", "string":
		return parseTypeName(under, namespace), nil
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
		// Treat unrecognized type names as named-type references (matches
		// parseObjectJSON's fallback for non-keyword types).
		ref := parseTypeName(under, namespace)
		if _, isRef := ref.(Ref); isRef {
			return ref, nil
		}
		return nil, UnknownTypeError{Type: under}
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
