// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical

import (
	"testing"

	"github.com/z5labs/avro-go/idl"

	"github.com/stretchr/testify/require"
)

func TestSchemaFrom(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    *idl.Schema
		expected []Schema
	}{
		{
			name: "null primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "null"},
			},
			expected: []Schema{PrimitiveSchema(Null)},
		},
		{
			name: "boolean primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "boolean"},
			},
			expected: []Schema{PrimitiveSchema(Boolean)},
		},
		{
			name: "int primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "int"},
			},
			expected: []Schema{PrimitiveSchema(Int)},
		},
		{
			name: "long primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "long"},
			},
			expected: []Schema{PrimitiveSchema(Long)},
		},
		{
			name: "float primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "float"},
			},
			expected: []Schema{PrimitiveSchema(Float)},
		},
		{
			name: "double primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "double"},
			},
			expected: []Schema{PrimitiveSchema(Double)},
		},
		{
			name: "bytes primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "bytes"},
			},
			expected: []Schema{PrimitiveSchema(Bytes)},
		},
		{
			name: "string primitive",
			input: &idl.Schema{
				Type: idl.Ident{Value: "string"},
			},
			expected: []Schema{PrimitiveSchema(String)},
		},
		{
			name: "record with schema namespace",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name: "Person",
					Fields: []*idl.Field{
						{Name: "name", Type: idl.Ident{Value: "string"}},
						{Name: "age", Type: idl.Ident{Value: "int"}},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
						{Name: "age", Type: PrimitiveSchema(Int)},
					},
				}),
			},
		},
		{
			name: "record with type namespace override",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name:      "Person",
					Namespace: "org.other",
					Fields: []*idl.Field{
						{Name: "name", Type: idl.Ident{Value: "string"}},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "org.other.Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
					},
				}),
			},
		},
		{
			name: "record with already qualified name",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name: "com.full.Person",
					Fields: []*idl.Field{
						{Name: "name", Type: idl.Ident{Value: "string"}},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.full.Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
					},
				}),
			},
		},
		{
			name: "record with no namespace",
			input: &idl.Schema{
				Type: idl.Record{
					Name: "Person",
					Fields: []*idl.Field{
						{Name: "name", Type: idl.Ident{Value: "string"}},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
					},
				}),
			},
		},
		{
			name: "record with no fields",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name:   "Empty",
					Fields: []*idl.Field{},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name:   "com.example.Empty",
					Fields: []Field{},
				}),
			},
		},
		{
			name: "record with named type reference field",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name: "Person",
					Fields: []*idl.Field{
						{Name: "address", Type: idl.Ident{Value: "Address"}},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "address", Type: PrimitiveSchema(Primitive("com.example.Address"))},
					},
				}),
			},
		},
		{
			name: "record with already qualified named type reference",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name: "Person",
					Fields: []*idl.Field{
						{Name: "address", Type: idl.Ident{Value: "org.other.Address"}},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "address", Type: PrimitiveSchema(Primitive("org.other.Address"))},
					},
				}),
			},
		},
		{
			name: "enum",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Enum{
					Name: "Color",
					Values: []*idl.Ident{
						{Value: "RED"},
						{Value: "GREEN"},
						{Value: "BLUE"},
					},
				},
			},
			expected: []Schema{
				EnumSchema(Enum{
					Name:    "com.example.Color",
					Symbols: []string{"RED", "GREEN", "BLUE"},
				}),
			},
		},
		{
			name: "enum with type namespace override",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Enum{
					Name:      "Color",
					Namespace: "org.other",
					Values: []*idl.Ident{
						{Value: "RED"},
					},
				},
			},
			expected: []Schema{
				EnumSchema(Enum{
					Name:    "org.other.Color",
					Symbols: []string{"RED"},
				}),
			},
		},
		{
			name: "array of primitives",
			input: &idl.Schema{
				Type: idl.Array{
					Items: idl.Ident{Value: "string"},
				},
			},
			expected: []Schema{
				ArraySchema(Array{Items: PrimitiveSchema(String)}),
			},
		},
		{
			name: "array of named type",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Array{
					Items: idl.Ident{Value: "Person"},
				},
			},
			expected: []Schema{
				ArraySchema(Array{Items: PrimitiveSchema(Primitive("com.example.Person"))}),
			},
		},
		{
			name: "map of primitives",
			input: &idl.Schema{
				Type: idl.Map{
					Values: &idl.Ident{Value: "int"},
				},
			},
			expected: []Schema{
				MapSchema(Map{Values: PrimitiveSchema(Int)}),
			},
		},
		{
			name: "map of named type",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Map{
					Values: &idl.Ident{Value: "Person"},
				},
			},
			expected: []Schema{
				MapSchema(Map{Values: PrimitiveSchema(Primitive("com.example.Person"))}),
			},
		},
		{
			name: "union",
			input: &idl.Schema{
				Type: idl.Union{
					Types: []idl.Type{
						idl.Ident{Value: "null"},
						idl.Ident{Value: "string"},
					},
				},
			},
			expected: []Schema{
				UnionSchema(Union{
					PrimitiveSchema(Null),
					PrimitiveSchema(String),
				}),
			},
		},
		{
			name: "union with named type",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Union{
					Types: []idl.Type{
						idl.Ident{Value: "null"},
						idl.Ident{Value: "Person"},
					},
				},
			},
			expected: []Schema{
				UnionSchema(Union{
					PrimitiveSchema(Null),
					PrimitiveSchema(Primitive("com.example.Person")),
				}),
			},
		},
		{
			name: "fixed",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Fixed{
					Name: "MD5",
					Size: 16,
				},
			},
			expected: []Schema{
				FixedSchema(Fixed{Name: "com.example.MD5", Size: 16}),
			},
		},
		{
			name: "fixed with type namespace override",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Fixed{
					Name:      "MD5",
					Namespace: "org.other",
					Size:      16,
				},
			},
			expected: []Schema{
				FixedSchema(Fixed{Name: "org.other.MD5", Size: 16}),
			},
		},
		{
			name: "multiple types",
			input: &idl.Schema{
				Namespace: "com.example",
				Types: []idl.Type{
					idl.Record{
						Name: "Person",
						Fields: []*idl.Field{
							{Name: "name", Type: idl.Ident{Value: "string"}},
						},
					},
					idl.Enum{
						Name: "Color",
						Values: []*idl.Ident{
							{Value: "RED"},
						},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
					},
				}),
				EnumSchema(Enum{
					Name:    "com.example.Color",
					Symbols: []string{"RED"},
				}),
			},
		},
		{
			name: "record with array field",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name: "Person",
					Fields: []*idl.Field{
						{
							Name: "tags",
							Type: idl.Array{Items: idl.Ident{Value: "string"}},
						},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "tags", Type: ArraySchema(Array{Items: PrimitiveSchema(String)})},
					},
				}),
			},
		},
		{
			name: "record with union field",
			input: &idl.Schema{
				Namespace: "com.example",
				Type: idl.Record{
					Name: "Person",
					Fields: []*idl.Field{
						{
							Name: "middle_name",
							Type: idl.Union{
								Types: []idl.Type{
									idl.Ident{Value: "null"},
									idl.Ident{Value: "string"},
								},
							},
						},
					},
				},
			},
			expected: []Schema{
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{
							Name: "middle_name",
							Type: UnionSchema(Union{
								PrimitiveSchema(Null),
								PrimitiveSchema(String),
							}),
						},
					},
				}),
			},
		},
		{
			name:     "empty schema with no type or types",
			input:    &idl.Schema{},
			expected: []Schema{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := SchemaFrom(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestSchemaFrom_nil_schema(t *testing.T) {
	t.Parallel()

	_, err := SchemaFrom(nil)

	require.Error(t, err)
}
