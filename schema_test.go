// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaInterface(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema Schema
	}{
		{name: "null", schema: Null{}},
		{name: "boolean", schema: Boolean{}},
		{name: "int", schema: Int{}},
		{name: "long", schema: Long{}},
		{name: "float", schema: Float{}},
		{name: "double", schema: Double{}},
		{name: "bytes", schema: Bytes{}},
		{name: "string", schema: String{}},
		{name: "ref", schema: Ref{Name: "Foo", Namespace: "com.example"}},
		{name: "record", schema: Record{Name: "R", Fields: []*Field{{Name: "f", Type: Int{}}}}},
		{name: "enum", schema: Enum{Name: "E", Symbols: []string{"A", "B"}}},
		{name: "array", schema: Array{Items: Long{}}},
		{name: "map", schema: Map{Values: String{}}},
		{name: "union", schema: Union{Types: []Schema{Null{}, Int{}}}},
		{name: "fixed", schema: Fixed{Name: "F", Size: 4}},
		{name: "decimal over bytes", schema: Decimal{Precision: 9, Scale: 2, Underlying: Bytes{}}},
		{name: "uuid", schema: UUID{}},
		{name: "date", schema: Date{}},
		{name: "time millis", schema: TimeMillis{}},
		{name: "time micros", schema: TimeMicros{}},
		{name: "timestamp millis", schema: TimestampMillis{}},
		{name: "timestamp micros", schema: TimestampMicros{}},
		{name: "timestamp nanos", schema: TimestampNanos{}},
		{name: "local timestamp millis", schema: LocalTimestampMillis{}},
		{name: "local timestamp micros", schema: LocalTimestampMicros{}},
		{name: "local timestamp nanos", schema: LocalTimestampNanos{}},
		{name: "duration", schema: Duration{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NotNil(t, tc.schema)
		})
	}
}

func TestRecordFields(t *testing.T) {
	t.Parallel()

	r := Record{
		Name:      "User",
		Namespace: "com.example",
		Fields: []*Field{
			{Name: "id", Type: Long{}, Order: OrderAscending},
			{Name: "email", Type: Union{Types: []Schema{Null{}, String{}}}, HasDefault: true, Default: nil},
			{Name: "tags", Type: Array{Items: String{}}, HasDefault: true, Default: []any{}},
		},
	}

	require.Equal(t, "User", r.Name)
	require.Len(t, r.Fields, 3)
	require.Equal(t, OrderAscending, r.Fields[0].Order)
}
