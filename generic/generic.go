// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

// Package generic provides Avro values that can be encoded and decoded at
// runtime using only an avro.Schema, without defining a Go struct per record.
//
// Values do not carry their schema; the schema is supplied to NewEncoder and
// NewDecoder and all validation is performed there.
package generic

import "math/big"

// Value is the interface implemented by all generic Avro values. The marker
// method is unexported so the type set is closed to this package.
type Value interface {
	value()
}

// Null is the Avro null value.
type Null struct{}

func (Null) value() {}

// Bool is the Avro boolean value.
type Bool bool

func (Bool) value() {}

// Int is the Avro 32-bit integer value.
type Int int32

func (Int) value() {}

// Long is the Avro 64-bit integer value.
type Long int64

func (Long) value() {}

// Float is the Avro 32-bit float value.
type Float float32

func (Float) value() {}

// Double is the Avro 64-bit float value.
type Double float64

func (Double) value() {}

// Bytes is the Avro variable-length byte sequence value.
type Bytes []byte

func (Bytes) value() {}

// String is the Avro string value.
type String string

func (String) value() {}

// Field is a single named field within a Record value.
type Field struct {
	Name  string
	Value Value
}

// Record is the Avro record value. Field order must match the schema's field
// order at encode time.
type Record struct {
	Fields []Field
}

func (Record) value() {}

// Enum is the Avro enum value.
type Enum struct {
	Symbol string
}

func (Enum) value() {}

// Array is the Avro array value.
type Array []Value

func (Array) value() {}

// Map is the Avro map value.
type Map map[string]Value

func (Map) value() {}

// Union is the Avro union value. Index selects the branch in the union
// schema's Types slice.
type Union struct {
	Index int
	Value Value
}

func (Union) value() {}

// Fixed is the Avro fixed-length byte sequence value.
type Fixed []byte

func (Fixed) value() {}

// Decimal is the Avro "decimal" logical-type value. Unscaled holds the
// unscaled integer; the scale is supplied by the schema.
type Decimal struct {
	Unscaled *big.Int
}

func (Decimal) value() {}

// UUID is the Avro "uuid" logical-type value (RFC 4122 textual form).
type UUID string

func (UUID) value() {}

// Date is the Avro "date" logical-type value: days since the Unix epoch.
type Date int32

func (Date) value() {}

// TimeMillis is the Avro "time-millis" logical-type value: milliseconds after
// midnight.
type TimeMillis int32

func (TimeMillis) value() {}

// TimeMicros is the Avro "time-micros" logical-type value: microseconds after
// midnight.
type TimeMicros int64

func (TimeMicros) value() {}

// TimestampMillis is the Avro "timestamp-millis" logical-type value:
// milliseconds since the Unix epoch in UTC.
type TimestampMillis int64

func (TimestampMillis) value() {}

// TimestampMicros is the Avro "timestamp-micros" logical-type value:
// microseconds since the Unix epoch in UTC.
type TimestampMicros int64

func (TimestampMicros) value() {}

// TimestampNanos is the Avro "timestamp-nanos" logical-type value:
// nanoseconds since the Unix epoch in UTC.
type TimestampNanos int64

func (TimestampNanos) value() {}

// LocalTimestampMillis is the Avro "local-timestamp-millis" logical-type
// value.
type LocalTimestampMillis int64

func (LocalTimestampMillis) value() {}

// LocalTimestampMicros is the Avro "local-timestamp-micros" logical-type
// value.
type LocalTimestampMicros int64

func (LocalTimestampMicros) value() {}

// LocalTimestampNanos is the Avro "local-timestamp-nanos" logical-type value.
type LocalTimestampNanos int64

func (LocalTimestampNanos) value() {}

// Duration is the Avro "duration" logical-type value. The wire form is three
// little-endian unsigned 32-bit integers: months, days, and milliseconds.
type Duration struct {
	Months uint32
	Days   uint32
	Millis uint32
}

func (Duration) value() {}
