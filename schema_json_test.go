// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchema_MarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema Schema
		want   string
	}{
		{name: "null", schema: Null{}, want: `"null"`},
		{name: "boolean", schema: Boolean{}, want: `"boolean"`},
		{name: "int", schema: Int{}, want: `"int"`},
		{name: "long", schema: Long{}, want: `"long"`},
		{name: "float", schema: Float{}, want: `"float"`},
		{name: "double", schema: Double{}, want: `"double"`},
		{name: "bytes", schema: Bytes{}, want: `"bytes"`},
		{name: "string", schema: String{}, want: `"string"`},
		{
			name:   "ref unqualified",
			schema: Ref{Name: "Foo"},
			want:   `"Foo"`,
		},
		{
			name:   "ref qualified",
			schema: Ref{Name: "Foo", Namespace: "com.example"},
			want:   `"com.example.Foo"`,
		},
		{
			name: "record",
			schema: Record{
				Name:      "User",
				Namespace: "com.example",
				Fields: []*Field{
					{Name: "id", Type: Long{}},
					{Name: "name", Type: String{}},
				},
			},
			want: `{"type":"record","name":"User","namespace":"com.example","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`,
		},
		{
			name:   "enum",
			schema: Enum{Name: "Color", Symbols: []string{"RED", "GREEN", "BLUE"}},
			want:   `{"type":"enum","name":"Color","symbols":["RED","GREEN","BLUE"]}`,
		},
		{
			name:   "array",
			schema: Array{Items: Long{}},
			want:   `{"type":"array","items":"long"}`,
		},
		{
			name:   "map",
			schema: Map{Values: String{}},
			want:   `{"type":"map","values":"string"}`,
		},
		{
			name:   "union",
			schema: Union{Types: []Schema{Null{}, String{}}},
			want:   `["null","string"]`,
		},
		{
			name:   "fixed",
			schema: Fixed{Name: "MD5", Size: 16},
			want:   `{"type":"fixed","name":"MD5","size":16}`,
		},
		{
			name:   "decimal over bytes",
			schema: Decimal{Precision: 9, Scale: 2, Underlying: Bytes{}},
			want:   `{"type":"bytes","logicalType":"decimal","precision":9,"scale":2}`,
		},
		{
			name:   "decimal over fixed",
			schema: Decimal{Precision: 9, Scale: 2, Underlying: Fixed{Name: "Dec", Size: 8}},
			want:   `{"type":"fixed","logicalType":"decimal","name":"Dec","size":8,"precision":9,"scale":2}`,
		},
		{name: "uuid", schema: UUID{}, want: `{"type":"string","logicalType":"uuid"}`},
		{name: "date", schema: Date{}, want: `{"type":"int","logicalType":"date"}`},
		{name: "time-millis", schema: TimeMillis{}, want: `{"type":"int","logicalType":"time-millis"}`},
		{name: "time-micros", schema: TimeMicros{}, want: `{"type":"long","logicalType":"time-micros"}`},
		{name: "timestamp-millis", schema: TimestampMillis{}, want: `{"type":"long","logicalType":"timestamp-millis"}`},
		{name: "timestamp-micros", schema: TimestampMicros{}, want: `{"type":"long","logicalType":"timestamp-micros"}`},
		{name: "timestamp-nanos", schema: TimestampNanos{}, want: `{"type":"long","logicalType":"timestamp-nanos"}`},
		{name: "local-timestamp-millis", schema: LocalTimestampMillis{}, want: `{"type":"long","logicalType":"local-timestamp-millis"}`},
		{name: "local-timestamp-micros", schema: LocalTimestampMicros{}, want: `{"type":"long","logicalType":"local-timestamp-micros"}`},
		{name: "local-timestamp-nanos", schema: LocalTimestampNanos{}, want: `{"type":"long","logicalType":"local-timestamp-nanos"}`},
		{name: "duration", schema: Duration{}, want: `{"type":"fixed","name":"duration","size":12,"logicalType":"duration"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.schema)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(got))
		})
	}
}

func TestParseJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		json string
		want Schema
	}{
		{name: "null", json: `"null"`, want: Null{}},
		{name: "string", json: `"string"`, want: String{}},
		{name: "ref qualified", json: `"com.example.Foo"`, want: Ref{Namespace: "com.example", Name: "Foo"}},
		{name: "ref unqualified", json: `"Foo"`, want: Ref{Name: "Foo"}},
		{
			name: "record",
			json: `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`,
			want: Record{
				Name:   "User",
				Fields: []*Field{{Name: "id", Type: Long{}}},
			},
		},
		{
			name: "enum",
			json: `{"type":"enum","name":"Color","symbols":["RED","GREEN"]}`,
			want: Enum{Name: "Color", Symbols: []string{"RED", "GREEN"}},
		},
		{name: "array", json: `{"type":"array","items":"long"}`, want: Array{Items: Long{}}},
		{name: "map", json: `{"type":"map","values":"string"}`, want: Map{Values: String{}}},
		{name: "union", json: `["null","string"]`, want: Union{Types: []Schema{Null{}, String{}}}},
		{name: "fixed", json: `{"type":"fixed","name":"MD5","size":16}`, want: Fixed{Name: "MD5", Size: 16}},
		{
			name: "decimal over bytes",
			json: `{"type":"bytes","logicalType":"decimal","precision":9,"scale":2}`,
			want: Decimal{Precision: 9, Scale: 2, Underlying: Bytes{}},
		},
		{
			name: "decimal over fixed",
			json: `{"type":"fixed","logicalType":"decimal","name":"Dec","size":8,"precision":9,"scale":2}`,
			want: Decimal{Precision: 9, Scale: 2, Underlying: Fixed{Name: "Dec", Size: 8}},
		},
		{name: "uuid", json: `{"type":"string","logicalType":"uuid"}`, want: UUID{}},
		{name: "date", json: `{"type":"int","logicalType":"date"}`, want: Date{}},
		{name: "duration", json: `{"type":"fixed","name":"duration","size":12,"logicalType":"duration"}`, want: Duration{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseJSON([]byte(tc.json))
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSchemaJSON_RoundTrip(t *testing.T) {
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
		{name: "ref qualified", schema: Ref{Namespace: "com.example", Name: "Foo"}},
		{
			name: "record with nested types",
			schema: Record{
				Name:      "User",
				Namespace: "com.example",
				Fields: []*Field{
					{Name: "id", Type: Long{}},
					{Name: "tags", Type: Array{Items: String{}}},
					{Name: "email", Type: Union{Types: []Schema{Null{}, String{}}}},
					{Name: "ref", Type: Ref{Name: "Other", Namespace: "com.example"}},
				},
			},
		},
		{
			name: "enum with default",
			schema: Enum{
				Name:    "Color",
				Symbols: []string{"RED", "GREEN", "BLUE"},
				Default: "RED",
			},
		},
		{name: "array of long", schema: Array{Items: Long{}}},
		{name: "map of string", schema: Map{Values: String{}}},
		{name: "union", schema: Union{Types: []Schema{Null{}, Long{}, String{}}}},
		{name: "fixed", schema: Fixed{Name: "MD5", Namespace: "com.example", Size: 16}},
		{name: "decimal over bytes", schema: Decimal{Precision: 9, Scale: 2, Underlying: Bytes{}}},
		{
			name: "decimal over fixed",
			schema: Decimal{
				Precision:  20,
				Scale:      4,
				Underlying: Fixed{Name: "Dec", Namespace: "com.example", Size: 12},
			},
		},
		{name: "uuid", schema: UUID{}},
		{name: "date", schema: Date{}},
		{name: "time-millis", schema: TimeMillis{}},
		{name: "time-micros", schema: TimeMicros{}},
		{name: "timestamp-millis", schema: TimestampMillis{}},
		{name: "timestamp-micros", schema: TimestampMicros{}},
		{name: "timestamp-nanos", schema: TimestampNanos{}},
		{name: "local-timestamp-millis", schema: LocalTimestampMillis{}},
		{name: "local-timestamp-micros", schema: LocalTimestampMicros{}},
		{name: "local-timestamp-nanos", schema: LocalTimestampNanos{}},
		{name: "duration", schema: Duration{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tc.schema)
			require.NoError(t, err)

			got, err := ParseJSON(data)
			require.NoError(t, err)
			require.Equal(t, tc.schema, got)
		})
	}
}

func TestParseJSON_Errors(t *testing.T) {
	t.Parallel()

	t.Run("missing record name", func(t *testing.T) {
		t.Parallel()
		_, err := ParseJSON([]byte(`{"type":"record","fields":[]}`))
		var got MissingFieldError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "record", got.Type)
		require.Equal(t, "name", got.Field)
	})

	t.Run("missing record fields", func(t *testing.T) {
		t.Parallel()
		_, err := ParseJSON([]byte(`{"type":"record","name":"R"}`))
		var got MissingFieldError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "fields", got.Field)
	})

	t.Run("unknown logical type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseJSON([]byte(`{"type":"int","logicalType":"bogus"}`))
		var got UnknownLogicalTypeError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "bogus", got.LogicalType)
	})

	t.Run("missing fixed size", func(t *testing.T) {
		t.Parallel()
		_, err := ParseJSON([]byte(`{"type":"fixed","name":"F"}`))
		var got MissingFieldError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "size", got.Field)
	})

	t.Run("missing decimal precision", func(t *testing.T) {
		t.Parallel()
		_, err := ParseJSON([]byte(`{"type":"bytes","logicalType":"decimal","scale":2}`))
		var got MissingFieldError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "decimal", got.Type)
		require.Equal(t, "precision", got.Field)
	})

	t.Run("invalid order string", func(t *testing.T) {
		t.Parallel()
		_, err := ParseJSON([]byte(`{"type":"record","name":"R","fields":[{"name":"a","type":"int","order":"sideways"}]}`))
		var got InvalidOrderStringError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "sideways", got.Value)
	})
}

// TestParseJSON_NamespaceInheritance verifies that nested named types without
// an explicit "namespace" inherit the enclosing record's namespace, so Refs
// resolve consistently regardless of where the type is defined.
func TestParseJSON_NamespaceInheritance(t *testing.T) {
	t.Parallel()

	src := `{
        "type": "record",
        "name": "Outer",
        "namespace": "com.example",
        "fields": [
            {"name": "color", "type": {"type": "enum", "name": "Color", "symbols": ["RED"]}},
            {"name": "hash",  "type": {"type": "fixed", "name": "MD5", "size": 16}}
        ]
    }`
	got, err := ParseJSON([]byte(src))
	require.NoError(t, err)

	rec := got.(Record)
	enum := rec.Fields[0].Type.(Enum)
	fixed := rec.Fields[1].Type.(Fixed)
	require.Equal(t, "com.example", enum.Namespace)
	require.Equal(t, "com.example", fixed.Namespace)
}

func TestField_DefaultAndOrderRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		field *Field
	}{
		{
			name:  "default string",
			field: &Field{Name: "x", Type: String{}, Default: "hello"},
		},
		{
			name:  "default null",
			field: &Field{Name: "x", Type: Union{Types: []Schema{Null{}, String{}}}, Default: nil},
		},
		{
			name:  "order descending",
			field: &Field{Name: "x", Type: Long{}, Order: OrderDescending},
		},
		{
			name:  "order ignore with default",
			field: &Field{Name: "x", Type: Boolean{}, Default: true, Order: OrderIgnore},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := Record{Name: "R", Fields: []*Field{tc.field}}
			data, err := json.Marshal(schema)
			require.NoError(t, err)

			got, err := ParseJSON(data)
			require.NoError(t, err)
			rec := got.(Record)
			require.Equal(t, tc.field.Name, rec.Fields[0].Name)
			require.Equal(t, tc.field.Order, rec.Fields[0].Order)
			require.Equal(t, tc.field.Default, rec.Fields[0].Default)
		})
	}
}
