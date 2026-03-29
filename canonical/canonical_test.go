// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchema_MarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		schema   Schema
		expected string
	}{
		{
			name:     "null primitive",
			schema:   PrimitiveSchema(Null),
			expected: `"null"`,
		},
		{
			name:     "boolean primitive",
			schema:   PrimitiveSchema(Boolean),
			expected: `"boolean"`,
		},
		{
			name:     "int primitive",
			schema:   PrimitiveSchema(Int),
			expected: `"int"`,
		},
		{
			name:     "long primitive",
			schema:   PrimitiveSchema(Long),
			expected: `"long"`,
		},
		{
			name:     "float primitive",
			schema:   PrimitiveSchema(Float),
			expected: `"float"`,
		},
		{
			name:     "double primitive",
			schema:   PrimitiveSchema(Double),
			expected: `"double"`,
		},
		{
			name:     "bytes primitive",
			schema:   PrimitiveSchema(Bytes),
			expected: `"bytes"`,
		},
		{
			name:     "string primitive",
			schema:   PrimitiveSchema(String),
			expected: `"string"`,
		},
		{
			name: "record with primitive fields",
			schema: RecordSchema(Record{
				Name: "com.example.Person",
				Fields: []Field{
					{Name: "name", Type: PrimitiveSchema(String)},
					{Name: "age", Type: PrimitiveSchema(Int)},
				},
			}),
			expected: `{"name":"com.example.Person","type":"record","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}`,
		},
		{
			name: "record with no fields",
			schema: RecordSchema(Record{
				Name:   "com.example.Empty",
				Fields: []Field{},
			}),
			expected: `{"name":"com.example.Empty","type":"record","fields":[]}`,
		},
		{
			name: "record with nested record field",
			schema: RecordSchema(Record{
				Name: "com.example.Outer",
				Fields: []Field{
					{
						Name: "inner",
						Type: RecordSchema(Record{
							Name: "com.example.Inner",
							Fields: []Field{
								{Name: "value", Type: PrimitiveSchema(Long)},
							},
						}),
					},
				},
			}),
			expected: `{"name":"com.example.Outer","type":"record","fields":[{"name":"inner","type":{"name":"com.example.Inner","type":"record","fields":[{"name":"value","type":"long"}]}}]}`,
		},
		{
			name: "enum",
			schema: EnumSchema(Enum{
				Name:    "com.example.Color",
				Symbols: []string{"RED", "GREEN", "BLUE"},
			}),
			expected: `{"name":"com.example.Color","type":"enum","symbols":["RED","GREEN","BLUE"]}`,
		},
		{
			name: "array of strings",
			schema: ArraySchema(Array{
				Items: PrimitiveSchema(String),
			}),
			expected: `{"type":"array","items":"string"}`,
		},
		{
			name: "array of records",
			schema: ArraySchema(Array{
				Items: RecordSchema(Record{
					Name: "com.example.Item",
					Fields: []Field{
						{Name: "id", Type: PrimitiveSchema(Long)},
					},
				}),
			}),
			expected: `{"type":"array","items":{"name":"com.example.Item","type":"record","fields":[{"name":"id","type":"long"}]}}`,
		},
		{
			name: "map of ints",
			schema: MapSchema(Map{
				Values: PrimitiveSchema(Int),
			}),
			expected: `{"type":"map","values":"int"}`,
		},
		{
			name: "union of null and string",
			schema: UnionSchema(Union{
				PrimitiveSchema(Null),
				PrimitiveSchema(String),
			}),
			expected: `["null","string"]`,
		},
		{
			name: "union with complex types",
			schema: UnionSchema(Union{
				PrimitiveSchema(Null),
				RecordSchema(Record{
					Name: "com.example.Event",
					Fields: []Field{
						{Name: "ts", Type: PrimitiveSchema(Long)},
					},
				}),
			}),
			expected: `["null",{"name":"com.example.Event","type":"record","fields":[{"name":"ts","type":"long"}]}]`,
		},
		{
			name: "fixed",
			schema: FixedSchema(Fixed{
				Name: "com.example.MD5",
				Size: 16,
			}),
			expected: `{"name":"com.example.MD5","type":"fixed","size":16}`,
		},
		{
			name: "record name with special characters",
			schema: RecordSchema(Record{
				Name: "com.example.\"Quoted\\Name\"\n",
				Fields: []Field{
					{Name: "field\twith\ttabs", Type: PrimitiveSchema(String)},
				},
			}),
			expected: `{"name":"com.example.\"Quoted\\Name\"\n","type":"record","fields":[{"name":"field\twith\ttabs","type":"string"}]}`,
		},
		{
			name: "enum symbols with backslash and quotes",
			schema: EnumSchema(Enum{
				Name:    "com.example.Escaped",
				Symbols: []string{"A\"B", "C\\D"},
			}),
			expected: `{"name":"com.example.Escaped","type":"enum","symbols":["A\"B","C\\D"]}`,
		},
		{
			name: "fixed name with control characters",
			schema: FixedSchema(Fixed{
				Name: "com.example.\b\f",
				Size: 4,
			}),
			expected: `{"name":"com.example.\b\f","type":"fixed","size":4}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tc.schema)

			require.NoError(t, err)
			require.Equal(t, tc.expected, string(b))
		})
	}
}

func TestSchema_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected Schema
	}{
		{
			name:     "null primitive",
			input:    `"null"`,
			expected: PrimitiveSchema(Null),
		},
		{
			name:     "boolean primitive",
			input:    `"boolean"`,
			expected: PrimitiveSchema(Boolean),
		},
		{
			name:     "int primitive",
			input:    `"int"`,
			expected: PrimitiveSchema(Int),
		},
		{
			name:     "long primitive",
			input:    `"long"`,
			expected: PrimitiveSchema(Long),
		},
		{
			name:     "float primitive",
			input:    `"float"`,
			expected: PrimitiveSchema(Float),
		},
		{
			name:     "double primitive",
			input:    `"double"`,
			expected: PrimitiveSchema(Double),
		},
		{
			name:     "bytes primitive",
			input:    `"bytes"`,
			expected: PrimitiveSchema(Bytes),
		},
		{
			name:     "string primitive",
			input:    `"string"`,
			expected: PrimitiveSchema(String),
		},
		{
			name:  "record with primitive fields",
			input: `{"name":"com.example.Person","type":"record","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}`,
			expected: RecordSchema(Record{
				Name: "com.example.Person",
				Fields: []Field{
					{Name: "name", Type: PrimitiveSchema(String)},
					{Name: "age", Type: PrimitiveSchema(Int)},
				},
			}),
		},
		{
			name:  "record with no fields",
			input: `{"name":"com.example.Empty","type":"record","fields":[]}`,
			expected: RecordSchema(Record{
				Name:   "com.example.Empty",
				Fields: []Field{},
			}),
		},
		{
			name:  "record with nested record field",
			input: `{"name":"com.example.Outer","type":"record","fields":[{"name":"inner","type":{"name":"com.example.Inner","type":"record","fields":[{"name":"value","type":"long"}]}}]}`,
			expected: RecordSchema(Record{
				Name: "com.example.Outer",
				Fields: []Field{
					{
						Name: "inner",
						Type: RecordSchema(Record{
							Name: "com.example.Inner",
							Fields: []Field{
								{Name: "value", Type: PrimitiveSchema(Long)},
							},
						}),
					},
				},
			}),
		},
		{
			name:  "enum",
			input: `{"name":"com.example.Color","type":"enum","symbols":["RED","GREEN","BLUE"]}`,
			expected: EnumSchema(Enum{
				Name:    "com.example.Color",
				Symbols: []string{"RED", "GREEN", "BLUE"},
			}),
		},
		{
			name:  "array of strings",
			input: `{"type":"array","items":"string"}`,
			expected: ArraySchema(Array{
				Items: PrimitiveSchema(String),
			}),
		},
		{
			name:  "map of ints",
			input: `{"type":"map","values":"int"}`,
			expected: MapSchema(Map{
				Values: PrimitiveSchema(Int),
			}),
		},
		{
			name:  "union of null and string",
			input: `["null","string"]`,
			expected: UnionSchema(Union{
				PrimitiveSchema(Null),
				PrimitiveSchema(String),
			}),
		},
		{
			name:  "union with complex types",
			input: `["null",{"name":"com.example.Event","type":"record","fields":[{"name":"ts","type":"long"}]}]`,
			expected: UnionSchema(Union{
				PrimitiveSchema(Null),
				RecordSchema(Record{
					Name: "com.example.Event",
					Fields: []Field{
						{Name: "ts", Type: PrimitiveSchema(Long)},
					},
				}),
			}),
		},
		{
			name:  "fixed",
			input: `{"name":"com.example.MD5","type":"fixed","size":16}`,
			expected: FixedSchema(Fixed{
				Name: "com.example.MD5",
				Size: 16,
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var s Schema
			err := json.Unmarshal([]byte(tc.input), &s)

			require.NoError(t, err)
			require.Equal(t, tc.expected, s)
		})
	}
}

func TestSchema_RoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema Schema
	}{
		{
			name:   "primitive",
			schema: PrimitiveSchema(String),
		},
		{
			name: "complex record",
			schema: RecordSchema(Record{
				Name: "com.example.Message",
				Fields: []Field{
					{Name: "id", Type: PrimitiveSchema(Long)},
					{Name: "body", Type: PrimitiveSchema(String)},
					{
						Name: "tags",
						Type: ArraySchema(Array{
							Items: PrimitiveSchema(String),
						}),
					},
					{
						Name: "metadata",
						Type: MapSchema(Map{
							Values: PrimitiveSchema(String),
						}),
					},
					{
						Name: "priority",
						Type: UnionSchema(Union{
							PrimitiveSchema(Null),
							EnumSchema(Enum{
								Name:    "com.example.Priority",
								Symbols: []string{"LOW", "MEDIUM", "HIGH"},
							}),
						}),
					},
				},
			}),
		},
		{
			name: "fixed",
			schema: FixedSchema(Fixed{
				Name: "com.example.SHA256",
				Size: 32,
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tc.schema)
			require.NoError(t, err)

			var s Schema
			err = json.Unmarshal(b, &s)

			require.NoError(t, err)
			require.Equal(t, tc.schema, s)
		})
	}
}

func TestSchema_Accessors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema Schema
		check  func(t *testing.T, s Schema)
	}{
		{
			name:   "primitive accessor returns value",
			schema: PrimitiveSchema(Int),
			check: func(t *testing.T, s Schema) {
				p, ok := s.Primitive()
				require.True(t, ok)
				require.Equal(t, Int, p)

				_, ok = s.Record()
				require.False(t, ok)
			},
		},
		{
			name: "record accessor returns value",
			schema: RecordSchema(Record{
				Name:   "com.example.Test",
				Fields: []Field{},
			}),
			check: func(t *testing.T, s Schema) {
				r, ok := s.Record()
				require.True(t, ok)
				require.Equal(t, "com.example.Test", r.Name)

				_, ok = s.Primitive()
				require.False(t, ok)
			},
		},
		{
			name: "enum accessor returns value",
			schema: EnumSchema(Enum{
				Name:    "com.example.Status",
				Symbols: []string{"ACTIVE", "INACTIVE"},
			}),
			check: func(t *testing.T, s Schema) {
				e, ok := s.Enum()
				require.True(t, ok)
				require.Equal(t, "com.example.Status", e.Name)
				require.Equal(t, []string{"ACTIVE", "INACTIVE"}, e.Symbols)

				_, ok = s.Record()
				require.False(t, ok)
			},
		},
		{
			name: "array accessor returns value",
			schema: ArraySchema(Array{
				Items: PrimitiveSchema(String),
			}),
			check: func(t *testing.T, s Schema) {
				a, ok := s.Array()
				require.True(t, ok)
				p, ok := a.Items.Primitive()
				require.True(t, ok)
				require.Equal(t, String, p)

				_, ok = s.Map()
				require.False(t, ok)
			},
		},
		{
			name: "map accessor returns value",
			schema: MapSchema(Map{
				Values: PrimitiveSchema(Int),
			}),
			check: func(t *testing.T, s Schema) {
				m, ok := s.Map()
				require.True(t, ok)
				p, ok := m.Values.Primitive()
				require.True(t, ok)
				require.Equal(t, Int, p)

				_, ok = s.Array()
				require.False(t, ok)
			},
		},
		{
			name: "union accessor returns value",
			schema: UnionSchema(Union{
				PrimitiveSchema(Null),
				PrimitiveSchema(String),
			}),
			check: func(t *testing.T, s Schema) {
				u, ok := s.Union()
				require.True(t, ok)
				require.Len(t, u, 2)

				_, ok = s.Fixed()
				require.False(t, ok)
			},
		},
		{
			name: "fixed accessor returns value",
			schema: FixedSchema(Fixed{
				Name: "com.example.Hash",
				Size: 16,
			}),
			check: func(t *testing.T, s Schema) {
				f, ok := s.Fixed()
				require.True(t, ok)
				require.Equal(t, "com.example.Hash", f.Name)
				require.Equal(t, 16, f.Size)

				_, ok = s.Union()
				require.False(t, ok)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.check(t, tc.schema)
		})
	}
}

func TestSchema_UnmarshalJSON_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "unexpected token",
			input:       "123",
			expectedErr: "canonical: unexpected JSON token",
		},
		{
			name:        "object missing type field",
			input:       `{"name":"com.example.Test"}`,
			expectedErr: `canonical: object schema missing "type" field`,
		},
		{
			name:        "unknown type",
			input:       `{"type":"unknown"}`,
			expectedErr: `canonical: unknown type "unknown"`,
		},
		{
			name:        "record missing name",
			input:       `{"type":"record","fields":[]}`,
			expectedErr: "canonical: record: missing name",
		},
		{
			name:        "record missing fields",
			input:       `{"type":"record","name":"com.example.Test"}`,
			expectedErr: "canonical: record: missing fields",
		},
		{
			name:        "record field missing name",
			input:       `{"type":"record","name":"com.example.Test","fields":[{"type":"string"}]}`,
			expectedErr: "canonical: record field: missing name",
		},
		{
			name:        "record field missing type",
			input:       `{"type":"record","name":"com.example.Test","fields":[{"name":"x"}]}`,
			expectedErr: "canonical: record field: missing type",
		},
		{
			name:        "enum missing name",
			input:       `{"type":"enum","symbols":["A"]}`,
			expectedErr: "canonical: enum: missing name",
		},
		{
			name:        "enum missing symbols",
			input:       `{"type":"enum","name":"com.example.E"}`,
			expectedErr: "canonical: enum: missing symbols",
		},
		{
			name:        "array missing items",
			input:       `{"type":"array"}`,
			expectedErr: "canonical: array: missing items",
		},
		{
			name:        "map missing values",
			input:       `{"type":"map"}`,
			expectedErr: "canonical: map: missing values",
		},
		{
			name:        "fixed missing name",
			input:       `{"type":"fixed","size":16}`,
			expectedErr: "canonical: fixed: missing name",
		},
		{
			name:        "fixed missing size",
			input:       `{"type":"fixed","name":"com.example.Hash"}`,
			expectedErr: "canonical: fixed: missing size",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var s Schema
			err := json.Unmarshal([]byte(tc.input), &s)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
