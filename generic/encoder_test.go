// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/z5labs/avro-go"
)

// encodeBytesFor is a small helper that runs an Encoder against v and returns
// the resulting wire bytes.
func encodeBytesFor(t *testing.T, s avro.Schema, v Value) []byte {
	t.Helper()
	enc, err := NewEncoder(s)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, enc.Encode(&buf, v))
	return buf.Bytes()
}

func TestEncoder_Primitives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		schema   avro.Schema
		value    Value
		expected []byte
	}{
		{name: "null", schema: avro.Null{}, value: Null{}, expected: nil},
		{name: "boolean true", schema: avro.Boolean{}, value: Bool(true), expected: []byte{0x01}},
		{name: "boolean false", schema: avro.Boolean{}, value: Bool(false), expected: []byte{0x00}},
		{name: "int 0", schema: avro.Int{}, value: Int(0), expected: []byte{0x00}},
		{name: "int 1", schema: avro.Int{}, value: Int(1), expected: []byte{0x02}},
		{name: "int -1", schema: avro.Int{}, value: Int(-1), expected: []byte{0x01}},
		{name: "long 64", schema: avro.Long{}, value: Long(64), expected: []byte{0x80, 0x01}},
		{name: "string abc", schema: avro.String{}, value: String("abc"), expected: []byte{0x06, 'a', 'b', 'c'}},
		{name: "bytes", schema: avro.Bytes{}, value: Bytes{0xde, 0xad}, expected: []byte{0x04, 0xde, 0xad}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := encodeBytesFor(t, tc.schema, tc.value)
			if tc.expected == nil {
				require.Empty(t, got)
				return
			}
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestEncoder_Record(t *testing.T) {
	t.Parallel()

	s := avro.Record{
		Name: "User",
		Fields: []*avro.Field{
			{Name: "id", Type: avro.Long{}},
			{Name: "name", Type: avro.String{}},
		},
	}
	v := Record{Fields: []Field{
		{Name: "id", Value: Long(1)},
		{Name: "name", Value: String("ada")},
	}}

	got := encodeBytesFor(t, s, v)
	require.Equal(t, []byte{0x02, 0x06, 'a', 'd', 'a'}, got)
}

func TestEncoder_Enum(t *testing.T) {
	t.Parallel()

	s := avro.Enum{Name: "Color", Symbols: []string{"RED", "GREEN", "BLUE"}}
	got := encodeBytesFor(t, s, Enum{Symbol: "GREEN"})
	require.Equal(t, []byte{0x02}, got)
}

func TestEncoder_Array(t *testing.T) {
	t.Parallel()

	s := avro.Array{Items: avro.Long{}}
	got := encodeBytesFor(t, s, Array{Long(1), Long(2)})
	// block of 2: zigzag count 4, items 0x02 0x04, terminator 0
	require.Equal(t, []byte{0x04, 0x02, 0x04, 0x00}, got)

	// empty array
	got = encodeBytesFor(t, s, Array{})
	require.Equal(t, []byte{0x00}, got)
}

func TestEncoder_Map(t *testing.T) {
	t.Parallel()

	s := avro.Map{Values: avro.Long{}}
	got := encodeBytesFor(t, s, Map{"a": Long(1)})
	require.Equal(t, []byte{0x02, 0x02, 'a', 0x02, 0x00}, got)

	got = encodeBytesFor(t, s, Map{})
	require.Equal(t, []byte{0x00}, got)
}

func TestEncoder_Union(t *testing.T) {
	t.Parallel()

	s := avro.Union{Types: []avro.Schema{avro.Null{}, avro.String{}}}
	got := encodeBytesFor(t, s, Union{Index: 1, Value: String("x")})
	require.Equal(t, []byte{0x02, 0x02, 'x'}, got)

	got = encodeBytesFor(t, s, Union{Index: 0, Value: Null{}})
	require.Equal(t, []byte{0x00}, got)
}

func TestEncoder_Fixed(t *testing.T) {
	t.Parallel()

	s := avro.Fixed{Name: "MD5", Size: 4}
	got := encodeBytesFor(t, s, Fixed{0x01, 0x02, 0x03, 0x04})
	require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, got)
}

func TestEncoder_Logicals(t *testing.T) {
	t.Parallel()

	t.Run("date", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, []byte{0x02}, encodeBytesFor(t, avro.Date{}, Date(1)))
	})
	t.Run("uuid", func(t *testing.T) {
		t.Parallel()
		s := "00000000-0000-0000-0000-000000000000"
		want := append([]byte{byte(len(s) << 1)}, []byte(s)...)
		require.Equal(t, want, encodeBytesFor(t, avro.UUID{}, UUID(s)))
	})
	t.Run("decimal over bytes", func(t *testing.T) {
		t.Parallel()
		s := avro.Decimal{Precision: 9, Scale: 2, Underlying: avro.Bytes{}}
		// 12345 -> 0x30 0x39, prepended with length 2 (zigzag 4 -> 0x04)
		got := encodeBytesFor(t, s, Decimal{Unscaled: big.NewInt(12345)})
		require.Equal(t, []byte{0x04, 0x30, 0x39}, got)
	})
	t.Run("decimal negative over bytes", func(t *testing.T) {
		t.Parallel()
		s := avro.Decimal{Precision: 9, Scale: 0, Underlying: avro.Bytes{}}
		got := encodeBytesFor(t, s, Decimal{Unscaled: big.NewInt(-1)})
		require.Equal(t, []byte{0x02, 0xff}, got)
	})
	t.Run("decimal over fixed", func(t *testing.T) {
		t.Parallel()
		s := avro.Decimal{Precision: 9, Scale: 2, Underlying: avro.Fixed{Name: "Dec", Size: 4}}
		got := encodeBytesFor(t, s, Decimal{Unscaled: big.NewInt(1)})
		require.Equal(t, []byte{0x00, 0x00, 0x00, 0x01}, got)
	})
	t.Run("duration", func(t *testing.T) {
		t.Parallel()
		got := encodeBytesFor(t, avro.Duration{}, Duration{Months: 1, Days: 2, Millis: 3})
		require.Equal(t, []byte{
			0x01, 0x00, 0x00, 0x00,
			0x02, 0x00, 0x00, 0x00,
			0x03, 0x00, 0x00, 0x00,
		}, got)
	})
}

func TestEncoder_Refs(t *testing.T) {
	t.Parallel()

	// A record that references itself: linked list.
	rec := avro.Record{
		Name: "Node",
		Fields: []*avro.Field{
			{Name: "v", Type: avro.Long{}},
			{Name: "next", Type: avro.Union{Types: []avro.Schema{avro.Null{}, avro.Ref{Name: "Node"}}}},
		},
	}
	v := Record{Fields: []Field{
		{Name: "v", Value: Long(1)},
		{Name: "next", Value: Union{Index: 1, Value: Record{Fields: []Field{
			{Name: "v", Value: Long(2)},
			{Name: "next", Value: Union{Index: 0, Value: Null{}}},
		}}}},
	}}
	got := encodeBytesFor(t, rec, v)
	// v=1, union 1 (Node), v=2, union 0 (null)
	require.Equal(t, []byte{0x02, 0x02, 0x04, 0x00}, got)
}

// ---- compile-time rejection tests ----

func TestNewEncoder_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema avro.Schema
		// substring expected in the error message
		errSub string
	}{
		{
			name:   "nil schema",
			schema: nil,
			errSub: "nil schema",
		},
		{
			name:   "unresolved ref",
			schema: avro.Ref{Name: "Missing"},
			errSub: "unresolved reference",
		},
		{
			name: "duplicate field names",
			schema: avro.Record{
				Name: "R",
				Fields: []*avro.Field{
					{Name: "x", Type: avro.Int{}},
					{Name: "x", Type: avro.Int{}},
				},
			},
			errSub: "duplicate field name",
		},
		{
			name:   "enum with no symbols",
			schema: avro.Enum{Name: "E"},
			errSub: "no symbols",
		},
		{
			name:   "enum default not in symbols",
			schema: avro.Enum{Name: "E", Symbols: []string{"A"}, Default: "B"},
			errSub: "default",
		},
		{
			name:   "enum duplicate symbols",
			schema: avro.Enum{Name: "E", Symbols: []string{"A", "A"}},
			errSub: "duplicate symbol",
		},
		{
			name:   "fixed zero size",
			schema: avro.Fixed{Name: "F", Size: 0},
			errSub: "size must be > 0",
		},
		{
			name:   "union with nested union",
			schema: avro.Union{Types: []avro.Schema{avro.Union{Types: []avro.Schema{avro.Int{}}}}},
			errSub: "itself a union",
		},
		{
			name:   "union duplicate unnamed",
			schema: avro.Union{Types: []avro.Schema{avro.Int{}, avro.Int{}}},
			errSub: "duplicate",
		},
		{
			name:   "decimal invalid precision",
			schema: avro.Decimal{Precision: 0, Scale: 0, Underlying: avro.Bytes{}},
			errSub: "precision",
		},
		{
			name:   "decimal invalid scale",
			schema: avro.Decimal{Precision: 4, Scale: 5, Underlying: avro.Bytes{}},
			errSub: "scale",
		},
		{
			name:   "decimal wrong underlying",
			schema: avro.Decimal{Precision: 4, Scale: 0, Underlying: avro.Int{}},
			errSub: "underlying",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewEncoder(tc.schema)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errSub)
		})
	}
}

// ---- runtime rejection tests ----

func TestEncode_RuntimeErrors(t *testing.T) {
	t.Parallel()

	t.Run("primitive type mismatch", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Int{})
		require.NoError(t, err)
		err = enc.Encode(&bytes.Buffer{}, String("nope"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "type mismatch")
	})

	t.Run("record field name mismatch", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Record{
			Name: "R",
			Fields: []*avro.Field{
				{Name: "a", Type: avro.Int{}},
				{Name: "b", Type: avro.Int{}},
			},
		})
		require.NoError(t, err)
		var buf bytes.Buffer
		err = enc.Encode(&buf, Record{Fields: []Field{
			{Name: "a", Value: Int(1)},
			{Name: "wrong", Value: Int(2)},
		}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected name")
	})

	t.Run("record wrong field count", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Record{
			Name: "R",
			Fields: []*avro.Field{
				{Name: "a", Type: avro.Int{}},
				{Name: "b", Type: avro.Int{}},
			},
		})
		require.NoError(t, err)
		err = enc.Encode(&bytes.Buffer{}, Record{Fields: []Field{{Name: "a", Value: Int(1)}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected 2 fields")
	})

	t.Run("union index out of range", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Union{Types: []avro.Schema{avro.Null{}, avro.Int{}}})
		require.NoError(t, err)
		err = enc.Encode(&bytes.Buffer{}, Union{Index: 5, Value: Int(0)})
		require.Error(t, err)
		require.Contains(t, err.Error(), "out of range")
	})

	t.Run("union branch type mismatch", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Union{Types: []avro.Schema{avro.Null{}, avro.Int{}}})
		require.NoError(t, err)
		var buf bytes.Buffer
		err = enc.Encode(&buf, Union{Index: 1, Value: String("x")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "type mismatch")
	})

	t.Run("logical/non-logical mismatch", func(t *testing.T) {
		t.Parallel()
		// Encoding a generic.Date into an avro.Int schema is rejected because
		// the schema's encoder expects a generic.Int.
		enc, err := NewEncoder(avro.Int{})
		require.NoError(t, err)
		err = enc.Encode(&bytes.Buffer{}, Date(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "type mismatch")
	})

	t.Run("enum unknown symbol", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Enum{Name: "E", Symbols: []string{"A", "B"}})
		require.NoError(t, err)
		err = enc.Encode(&bytes.Buffer{}, Enum{Symbol: "X"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not in schema")
	})

	t.Run("fixed size mismatch", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Fixed{Name: "F", Size: 4})
		require.NoError(t, err)
		err = enc.Encode(&bytes.Buffer{}, Fixed{0x01, 0x02})
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected 4 bytes")
	})

	t.Run("decimal precision overflow", func(t *testing.T) {
		t.Parallel()
		enc, err := NewEncoder(avro.Decimal{Precision: 4, Scale: 0, Underlying: avro.Bytes{}})
		require.NoError(t, err)
		// 10^4 exceeds 4 digits of precision.
		err = enc.Encode(&bytes.Buffer{}, Decimal{Unscaled: big.NewInt(10000)})
		require.Error(t, err)
		require.Contains(t, err.Error(), "precision")
	})
}
