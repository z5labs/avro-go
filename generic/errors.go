// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"errors"
	"fmt"

	"github.com/z5labs/avro-go"
)

// Sentinel errors for parameterless failures. They are returned directly so
// callers can assert with errors.Is.
var (
	// ErrNilSchema is returned by NewEncoder and NewDecoder when given a nil
	// schema.
	ErrNilSchema = errors.New("avro/generic: nil schema")

	// ErrEmptyUnion is returned when a Union schema declares no branches.
	ErrEmptyUnion = errors.New("avro/generic: union has no branches")

	// ErrNilArrayItems is returned when an Array schema's Items is nil.
	ErrNilArrayItems = errors.New("avro/generic: array has nil items schema")

	// ErrNilMapValues is returned when a Map schema's Values is nil.
	ErrNilMapValues = errors.New("avro/generic: map has nil values schema")

	// ErrNilDecimalUnscaled is returned when encoding a Decimal whose
	// Unscaled big.Int pointer is nil.
	ErrNilDecimalUnscaled = errors.New("avro/generic: decimal: nil Unscaled")
)

// ---- schema-level errors ----

// UnsupportedSchemaError reports a schema implementation the codec does not
// know how to compile.
type UnsupportedSchemaError struct{ Schema avro.Schema }

func (e UnsupportedSchemaError) Error() string {
	return fmt.Sprintf("avro/generic: unsupported schema type %T", e.Schema)
}

// MissingNameError reports a named schema (record/enum/fixed/field) whose
// Name field is empty. Kind names which named-type kind triggered the error.
type MissingNameError struct{ Kind string }

func (e MissingNameError) Error() string {
	return fmt.Sprintf("avro/generic: %s missing name", e.Kind)
}

// DuplicateNameError reports two named schemas registered under the same
// fully-qualified name.
type DuplicateNameError struct{ Name string }

func (e DuplicateNameError) Error() string {
	return fmt.Sprintf("avro/generic: duplicate named type %q", e.Name)
}

// UnresolvedReferenceError reports a Ref that does not match any previously
// registered named schema.
type UnresolvedReferenceError struct{ Name string }

func (e UnresolvedReferenceError) Error() string {
	return fmt.Sprintf("avro/generic: unresolved reference %q", e.Name)
}

// DuplicateFieldError reports a record schema with two fields sharing a name.
type DuplicateFieldError struct{ Record, Field string }

func (e DuplicateFieldError) Error() string {
	return fmt.Sprintf("avro/generic: record %q: duplicate field name %q", e.Record, e.Field)
}

// InvalidFieldError reports a malformed Field within a record schema (nil
// pointer, empty Name, or nil Type).
type InvalidFieldError struct {
	Record string
	Index  int
	Field  string // empty when the entry was nil or unnamed
	Reason string
}

func (e InvalidFieldError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("avro/generic: record %q: field %q: %s", e.Record, e.Field, e.Reason)
	}
	return fmt.Sprintf("avro/generic: record %q: field %d: %s", e.Record, e.Index, e.Reason)
}

// FieldCompileError wraps an error encountered while compiling a record
// field's nested schema, preserving the underlying cause for errors.Is /
// errors.As traversal.
type FieldCompileError struct {
	Record, Field string
	Err           error
}

func (e FieldCompileError) Error() string {
	return fmt.Sprintf("avro/generic: record %q field %q: %v", e.Record, e.Field, e.Err)
}

func (e FieldCompileError) Unwrap() error { return e.Err }

// ItemsCompileError wraps an error from compiling an array's Items schema.
type ItemsCompileError struct{ Err error }

func (e ItemsCompileError) Error() string { return fmt.Sprintf("avro/generic: array items: %v", e.Err) }
func (e ItemsCompileError) Unwrap() error { return e.Err }

// ValuesCompileError wraps an error from compiling a map's Values schema.
type ValuesCompileError struct{ Err error }

func (e ValuesCompileError) Error() string { return fmt.Sprintf("avro/generic: map values: %v", e.Err) }
func (e ValuesCompileError) Unwrap() error { return e.Err }

// BranchCompileError wraps an error from compiling one branch of a union.
type BranchCompileError struct {
	Index int
	Err   error
}

func (e BranchCompileError) Error() string {
	return fmt.Sprintf("avro/generic: union branch %d: %v", e.Index, e.Err)
}

func (e BranchCompileError) Unwrap() error { return e.Err }

// EmptySymbolsError reports an Enum schema with no symbols.
type EmptySymbolsError struct{ Enum string }

func (e EmptySymbolsError) Error() string {
	return fmt.Sprintf("avro/generic: enum %q has no symbols", e.Enum)
}

// DuplicateSymbolError reports an Enum schema with two identical symbols.
type DuplicateSymbolError struct{ Enum, Symbol string }

func (e DuplicateSymbolError) Error() string {
	return fmt.Sprintf("avro/generic: enum %q: duplicate symbol %q", e.Enum, e.Symbol)
}

// InvalidEnumDefaultError reports an Enum whose Default is not one of its
// declared symbols.
type InvalidEnumDefaultError struct{ Enum, Default string }

func (e InvalidEnumDefaultError) Error() string {
	return fmt.Sprintf("avro/generic: enum %q: default %q not in symbols", e.Enum, e.Default)
}

// InvalidFixedSizeError reports a Fixed schema with non-positive Size.
type InvalidFixedSizeError struct {
	Name string
	Size int
}

func (e InvalidFixedSizeError) Error() string {
	return fmt.Sprintf("avro/generic: fixed %q: size must be > 0, got %d", e.Name, e.Size)
}

// InvalidDurationSizeError reports a Duration schema whose Underlying.Size
// is set to a value other than 12 (the size required by the Avro spec).
type InvalidDurationSizeError struct{ Size int }

func (e InvalidDurationSizeError) Error() string {
	return fmt.Sprintf("avro/generic: duration: underlying size must be 12, got %d", e.Size)
}

// NilBranchError reports a nil schema in a Union's Types slice.
type NilBranchError struct{ Index int }

func (e NilBranchError) Error() string {
	return fmt.Sprintf("avro/generic: union branch %d is nil", e.Index)
}

// NestedUnionError reports a Union containing a Union as one of its branches.
type NestedUnionError struct{ Index int }

func (e NestedUnionError) Error() string {
	return fmt.Sprintf("avro/generic: union branch %d is itself a union", e.Index)
}

// DuplicateBranchError reports a Union with two branches that resolve to the
// same key (same primitive type, or same fully-qualified named type). Named
// reports whether the duplication is among named branches.
type DuplicateBranchError struct {
	Key   string
	Named bool
}

func (e DuplicateBranchError) Error() string {
	if e.Named {
		return fmt.Sprintf("avro/generic: union: duplicate named branch %q", e.Key)
	}
	return fmt.Sprintf("avro/generic: union: duplicate %s branch", e.Key)
}

// InvalidPrecisionError reports a Decimal schema with non-positive Precision.
type InvalidPrecisionError struct{ Precision int }

func (e InvalidPrecisionError) Error() string {
	return fmt.Sprintf("avro/generic: decimal: precision must be > 0, got %d", e.Precision)
}

// InvalidScaleError reports a Decimal schema with Scale outside [0, Precision].
type InvalidScaleError struct{ Precision, Scale int }

func (e InvalidScaleError) Error() string {
	return fmt.Sprintf("avro/generic: decimal: scale %d out of range [0, %d]", e.Scale, e.Precision)
}

// InvalidDecimalUnderlyingError reports a Decimal whose Underlying schema is
// neither Bytes nor Fixed.
type InvalidDecimalUnderlyingError struct{ Underlying avro.Schema }

func (e InvalidDecimalUnderlyingError) Error() string {
	return fmt.Sprintf("avro/generic: decimal: underlying must be bytes or fixed, got %T", e.Underlying)
}

// ---- runtime errors ----

// TypeMismatchError reports a Value whose Go type does not match the schema
// position currently being encoded.
type TypeMismatchError struct {
	Expected string
	Got      Value
}

func (e TypeMismatchError) Error() string {
	return fmt.Sprintf("avro/generic: type mismatch: schema expects %s, got %T", e.Expected, e.Got)
}

// FieldCountError reports a Record value with a different number of fields
// than the schema declared.
type FieldCountError struct {
	Record         string
	Expected, Got  int
}

func (e FieldCountError) Error() string {
	return fmt.Sprintf("avro/generic: record %q: expected %d fields, got %d", e.Record, e.Expected, e.Got)
}

// FieldNameError reports a Record value whose field name at Index does not
// match the schema's field at the same position.
type FieldNameError struct {
	Record         string
	Index          int
	Expected, Got  string
}

func (e FieldNameError) Error() string {
	return fmt.Sprintf("avro/generic: record %q: field %d: expected name %q, got %q", e.Record, e.Index, e.Expected, e.Got)
}

// IndexOutOfRangeError reports a Union or Enum index that falls outside the
// schema's range. Kind is "union" or "enum"; Name is set for enums (whose
// schema is named) and empty for unions.
type IndexOutOfRangeError struct {
	Kind  string
	Name  string
	Index int64
	Len   int
}

func (e IndexOutOfRangeError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("avro/generic: %s %q: index %d out of range [0, %d)", e.Kind, e.Name, e.Index, e.Len)
	}
	return fmt.Sprintf("avro/generic: %s: index %d out of range [0, %d)", e.Kind, e.Index, e.Len)
}

// UnknownSymbolError reports an Enum value whose Symbol is not one of the
// schema's declared symbols.
type UnknownSymbolError struct{ Enum, Symbol string }

func (e UnknownSymbolError) Error() string {
	return fmt.Sprintf("avro/generic: enum %q: symbol %q not in schema", e.Enum, e.Symbol)
}

// FixedSizeError reports a Fixed value whose byte length does not match the
// schema's Size.
type FixedSizeError struct {
	Name           string
	Expected, Got  int
}

func (e FixedSizeError) Error() string {
	return fmt.Sprintf("avro/generic: fixed %q: expected %d bytes, got %d", e.Name, e.Expected, e.Got)
}

// PrecisionOverflowError reports a Decimal value whose magnitude exceeds the
// schema's Precision digit budget.
type PrecisionOverflowError struct{ Precision int }

func (e PrecisionOverflowError) Error() string {
	return fmt.Sprintf("avro/generic: decimal: value exceeds precision %d", e.Precision)
}

// DecimalSizeError reports a Decimal whose two's-complement encoding does
// not fit in a fixed-backed schema's byte size.
type DecimalSizeError struct {
	Encoded   int
	FixedSize int
}

func (e DecimalSizeError) Error() string {
	return fmt.Sprintf("avro/generic: decimal: encoded %d bytes exceeds fixed size %d", e.Encoded, e.FixedSize)
}
