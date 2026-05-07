// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

// Schema represents an Avro schema. Concrete implementations model the JSON
// schema spec at https://avro.apache.org/docs/1.12.0/specification/.
//
// Schema values are pure data; all validation is performed by the codecs that
// consume them (e.g. github.com/z5labs/avro-go/generic.NewEncoder).
type Schema interface {
	schema()
}

// Null is the Avro "null" primitive schema.
type Null struct{}

func (Null) schema() {}

// Boolean is the Avro "boolean" primitive schema.
type Boolean struct{}

func (Boolean) schema() {}

// Int is the Avro "int" primitive schema (32-bit signed integer).
type Int struct{}

func (Int) schema() {}

// Long is the Avro "long" primitive schema (64-bit signed integer).
type Long struct{}

func (Long) schema() {}

// Float is the Avro "float" primitive schema (32-bit IEEE 754 float).
type Float struct{}

func (Float) schema() {}

// Double is the Avro "double" primitive schema (64-bit IEEE 754 float).
type Double struct{}

func (Double) schema() {}

// Bytes is the Avro "bytes" primitive schema (sequence of 8-bit bytes).
type Bytes struct{}

func (Bytes) schema() {}

// String is the Avro "string" primitive schema (Unicode character sequence).
type String struct{}

func (String) schema() {}

// Ref is a reference to a previously declared named schema (record, enum, or
// fixed). It is distinct from the primitive schemas so that name resolution
// is unambiguous.
type Ref struct {
	Name      string
	Namespace string
}

func (Ref) schema() {}

// Order describes the sort order of a record field.
type Order int

const (
	OrderAscending Order = iota
	OrderDescending
	OrderIgnore
)

// Field describes a single field within a Record.
type Field struct {
	Name    string
	Aliases []string
	Doc     string
	Type    Schema
	// Default is the field's default value as defined by the Avro spec.
	// HasDefault distinguishes "no default specified" (HasDefault=false) from
	// an explicit null default (HasDefault=true, Default=nil), which is needed
	// for unions whose first branch is null.
	Default    any
	HasDefault bool
	Order      Order
}

// Record is a named record schema.
type Record struct {
	Name      string
	Namespace string
	Doc       string
	Aliases   []string
	Fields    []*Field
}

func (Record) schema() {}

// Enum is a named enum schema.
type Enum struct {
	Name      string
	Namespace string
	Doc       string
	Aliases   []string
	Symbols   []string
	// Default is the symbol used when a reader encounters an unknown writer
	// symbol. An empty string means no default was specified.
	Default string
}

func (Enum) schema() {}

// Array is an array schema whose elements are of type Items.
type Array struct {
	Items Schema
}

func (Array) schema() {}

// Map is a map schema whose values are of type Values. Keys are always strings.
type Map struct {
	Values Schema
}

func (Map) schema() {}

// Union is a union schema. The order of Types matches the on-the-wire branch
// index used by Avro's binary encoding.
type Union struct {
	Types []Schema
}

func (Union) schema() {}

// Fixed is a named fixed-length byte sequence schema.
type Fixed struct {
	Name      string
	Namespace string
	Aliases   []string
	Size      int
}

func (Fixed) schema() {}

// Decimal is the Avro "decimal" logical type. It must be backed by either a
// Bytes or Fixed schema. Precision is the maximum number of decimal digits;
// Scale is the number of digits to the right of the decimal point and must
// satisfy 0 <= Scale <= Precision.
type Decimal struct {
	Precision  int
	Scale      int
	Underlying Schema
}

func (Decimal) schema() {}

// UUID is the Avro "uuid" logical type, backed by String.
type UUID struct{}

func (UUID) schema() {}

// Date is the Avro "date" logical type, backed by Int. It represents days
// since the Unix epoch (1970-01-01).
type Date struct{}

func (Date) schema() {}

// TimeMillis is the Avro "time-millis" logical type, backed by Int.
// Milliseconds after midnight, in the range [0, 86_400_000).
type TimeMillis struct{}

func (TimeMillis) schema() {}

// TimeMicros is the Avro "time-micros" logical type, backed by Long.
// Microseconds after midnight, in the range [0, 86_400_000_000).
type TimeMicros struct{}

func (TimeMicros) schema() {}

// TimestampMillis is the Avro "timestamp-millis" logical type, backed by Long.
// Milliseconds since the Unix epoch in UTC.
type TimestampMillis struct{}

func (TimestampMillis) schema() {}

// TimestampMicros is the Avro "timestamp-micros" logical type, backed by Long.
// Microseconds since the Unix epoch in UTC.
type TimestampMicros struct{}

func (TimestampMicros) schema() {}

// TimestampNanos is the Avro "timestamp-nanos" logical type, backed by Long.
// Nanoseconds since the Unix epoch in UTC.
type TimestampNanos struct{}

func (TimestampNanos) schema() {}

// LocalTimestampMillis is the Avro "local-timestamp-millis" logical type, backed
// by Long.
type LocalTimestampMillis struct{}

func (LocalTimestampMillis) schema() {}

// LocalTimestampMicros is the Avro "local-timestamp-micros" logical type, backed
// by Long.
type LocalTimestampMicros struct{}

func (LocalTimestampMicros) schema() {}

// LocalTimestampNanos is the Avro "local-timestamp-nanos" logical type, backed
// by Long.
type LocalTimestampNanos struct{}

func (LocalTimestampNanos) schema() {}

// Duration is the Avro "duration" logical type, backed by Fixed of size 12.
// The wire form is three little-endian unsigned 32-bit integers: months, days,
// and milliseconds.
type Duration struct{}

func (Duration) schema() {}
