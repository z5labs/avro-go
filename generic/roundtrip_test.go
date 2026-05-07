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

// roundTrip encodes v with s, then decodes and returns the decoded Value.
func roundTrip(t *testing.T, s avro.Schema, v Value) Value {
	t.Helper()
	enc, err := NewEncoder(s)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, enc.Encode(&buf, v))

	dec, err := NewDecoder(s)
	require.NoError(t, err)
	out, err := dec.Decode(&buf)
	require.NoError(t, err)
	return out
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema avro.Schema
		value  Value
	}{
		{name: "null", schema: avro.Null{}, value: Null{}},
		{name: "boolean", schema: avro.Boolean{}, value: Bool(true)},
		{name: "int", schema: avro.Int{}, value: Int(-123456)},
		{name: "long", schema: avro.Long{}, value: Long(1234567890123)},
		{name: "float", schema: avro.Float{}, value: Float(3.14)},
		{name: "double", schema: avro.Double{}, value: Double(2.718281828)},
		{name: "bytes", schema: avro.Bytes{}, value: Bytes{0xde, 0xad, 0xbe, 0xef}},
		{name: "string", schema: avro.String{}, value: String("hello, world")},
		{
			name: "record",
			schema: avro.Record{
				Name: "User",
				Fields: []*avro.Field{
					{Name: "id", Type: avro.Long{}},
					{Name: "name", Type: avro.String{}},
				},
			},
			value: Record{Fields: []Field{
				{Name: "id", Value: Long(42)},
				{Name: "name", Value: String("ada")},
			}},
		},
		{
			name:   "enum",
			schema: avro.Enum{Name: "Color", Symbols: []string{"RED", "GREEN", "BLUE"}},
			value:  Enum{Symbol: "GREEN"},
		},
		{
			name:   "array",
			schema: avro.Array{Items: avro.Long{}},
			value:  Array{Long(1), Long(2), Long(3)},
		},
		{
			name:   "empty array",
			schema: avro.Array{Items: avro.Long{}},
			value:  Array{},
		},
		{
			name:   "map",
			schema: avro.Map{Values: avro.Long{}},
			value:  Map{"a": Long(1), "b": Long(2)},
		},
		{
			name:   "empty map",
			schema: avro.Map{Values: avro.Long{}},
			value:  Map{},
		},
		{
			name:   "union null",
			schema: avro.Union{Types: []avro.Schema{avro.Null{}, avro.String{}}},
			value:  Union{Index: 0, Value: Null{}},
		},
		{
			name:   "union string",
			schema: avro.Union{Types: []avro.Schema{avro.Null{}, avro.String{}}},
			value:  Union{Index: 1, Value: String("hi")},
		},
		{
			name:   "fixed",
			schema: avro.Fixed{Name: "MD5", Size: 4},
			value:  Fixed{0x01, 0x02, 0x03, 0x04},
		},
		// logical types
		{
			name:   "decimal over bytes positive",
			schema: avro.Decimal{Precision: 9, Scale: 2, Underlying: avro.Bytes{}},
			value:  Decimal{Unscaled: big.NewInt(12345)},
		},
		{
			name:   "decimal over bytes negative",
			schema: avro.Decimal{Precision: 9, Scale: 2, Underlying: avro.Bytes{}},
			value:  Decimal{Unscaled: big.NewInt(-12345)},
		},
		{
			name:   "decimal over fixed",
			schema: avro.Decimal{Precision: 9, Scale: 2, Underlying: avro.Fixed{Name: "Dec", Size: 8}},
			value:  Decimal{Unscaled: big.NewInt(-1)},
		},
		{
			name:   "uuid",
			schema: avro.UUID{},
			value:  UUID("123e4567-e89b-12d3-a456-426614174000"),
		},
		{name: "date", schema: avro.Date{}, value: Date(19000)},
		{name: "time-millis", schema: avro.TimeMillis{}, value: TimeMillis(123456)},
		{name: "time-micros", schema: avro.TimeMicros{}, value: TimeMicros(123456789)},
		{name: "timestamp-millis", schema: avro.TimestampMillis{}, value: TimestampMillis(1700000000000)},
		{name: "timestamp-micros", schema: avro.TimestampMicros{}, value: TimestampMicros(1700000000000000)},
		{name: "timestamp-nanos", schema: avro.TimestampNanos{}, value: TimestampNanos(1700000000000000000)},
		{name: "local-timestamp-millis", schema: avro.LocalTimestampMillis{}, value: LocalTimestampMillis(1)},
		{name: "local-timestamp-micros", schema: avro.LocalTimestampMicros{}, value: LocalTimestampMicros(1)},
		{name: "local-timestamp-nanos", schema: avro.LocalTimestampNanos{}, value: LocalTimestampNanos(1)},
		{
			name:   "duration",
			schema: avro.Duration{},
			value:  Duration{Months: 12, Days: 31, Millis: 86399999},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := roundTrip(t, tc.schema, tc.value)
			// Decimal needs special compare because *big.Int identity differs.
			if d, ok := tc.value.(Decimal); ok {
				gd, ok := got.(Decimal)
				require.True(t, ok)
				require.Equal(t, 0, d.Unscaled.Cmp(gd.Unscaled))
				return
			}
			require.Equal(t, tc.value, got)
		})
	}
}

func TestRoundTrip_RecursiveRecord(t *testing.T) {
	t.Parallel()

	schema := avro.Record{
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

	got := roundTrip(t, schema, v)
	require.Equal(t, v, got)
}
