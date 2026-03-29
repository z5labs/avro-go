// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical

import (
	"strings"
	"testing"

	"github.com/z5labs/avro-go/idl"

	"github.com/stretchr/testify/require"
)

func TestSchemaFrom(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		src      string
		expected []Schema
	}{
		{
			name: "null primitive",
			src:  `schema null;`,
			expected: []Schema{
				PrimitiveSchema(Null),
			},
		},
		{
			name: "boolean primitive",
			src:  `schema boolean;`,
			expected: []Schema{
				PrimitiveSchema(Boolean),
			},
		},
		{
			name: "int primitive",
			src:  `schema int;`,
			expected: []Schema{
				PrimitiveSchema(Int),
			},
		},
		{
			name: "long primitive",
			src:  `schema long;`,
			expected: []Schema{
				PrimitiveSchema(Long),
			},
		},
		{
			name: "float primitive",
			src:  `schema float;`,
			expected: []Schema{
				PrimitiveSchema(Float),
			},
		},
		{
			name: "double primitive",
			src:  `schema double;`,
			expected: []Schema{
				PrimitiveSchema(Double),
			},
		},
		{
			name: "bytes primitive",
			src:  `schema bytes;`,
			expected: []Schema{
				PrimitiveSchema(Bytes),
			},
		},
		{
			name: "string primitive",
			src:  `schema string;`,
			expected: []Schema{
				PrimitiveSchema(String),
			},
		},
		{
			name: "record with schema namespace",
			src: `namespace com.example;
schema int;
record Person {
	string name;
	int age;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
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
			src: `namespace com.example;
schema int;
@namespace("org.other")
record Person {
	string name;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
				RecordSchema(Record{
					Name: "org.other.Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
					},
				}),
			},
		},
		{
			name: "record with named type reference field",
			src: `namespace com.example;
schema int;
record Person {
	Address address;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
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
			src: `namespace com.example;
schema int;
record Person {
	org.other.Address address;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "address", Type: PrimitiveSchema(Primitive("org.other.Address"))},
					},
				}),
			},
		},
		{
			name: "record with nullable field",
			src: `namespace com.example;
schema int;
record Person {
	string? middle_name;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
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
			name: "record with union field",
			src: `namespace com.example;
schema int;
record Person {
	union { null, string } middle_name;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
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
			name: "record with map field",
			src: `namespace com.example;
schema int;
record Person {
	map<string> tags;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
				RecordSchema(Record{
					Name: "com.example.Person",
					Fields: []Field{
						{Name: "tags", Type: MapSchema(Map{Values: PrimitiveSchema(String)})},
					},
				}),
			},
		},
		{
			name: "enum",
			src: `namespace com.example;
schema int;
enum Color {
	RED, GREEN, BLUE
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
				EnumSchema(Enum{
					Name:    "com.example.Color",
					Symbols: []string{"RED", "GREEN", "BLUE"},
				}),
			},
		},
		{
			name: "enum with type namespace override",
			src: `namespace com.example;
schema int;
@namespace("org.other")
enum Color {
	RED
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
				EnumSchema(Enum{
					Name:    "org.other.Color",
					Symbols: []string{"RED"},
				}),
			},
		},
		{
			name: "fixed",
			src: `namespace com.example;
schema int;
fixed MD5(16);`,
			expected: []Schema{
				PrimitiveSchema(Int),
				FixedSchema(Fixed{Name: "com.example.MD5", Size: 16}),
			},
		},
		{
			name: "fixed with type namespace override",
			src: `namespace com.example;
schema int;
@namespace("org.other")
fixed MD5(16);`,
			expected: []Schema{
				PrimitiveSchema(Int),
				FixedSchema(Fixed{Name: "org.other.MD5", Size: 16}),
			},
		},
		{
			name: "multiple types",
			src: `namespace com.example;
schema int;
record Person {
	string name;
}
enum Color {
	RED
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
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
			name: "map schema type",
			src:  `schema map<int>;`,
			expected: []Schema{
				MapSchema(Map{Values: PrimitiveSchema(Int)}),
			},
		},
		{
			name: "map of named type",
			src: `namespace com.example;
schema map<Person>;`,
			expected: []Schema{
				MapSchema(Map{Values: PrimitiveSchema(Primitive("com.example.Person"))}),
			},
		},
		{
			name: "union schema type",
			src:  `schema union { null, string };`,
			expected: []Schema{
				UnionSchema(Union{
					PrimitiveSchema(Null),
					PrimitiveSchema(String),
				}),
			},
		},
		{
			name: "union with named type",
			src: `namespace com.example;
schema union { null, Person };`,
			expected: []Schema{
				UnionSchema(Union{
					PrimitiveSchema(Null),
					PrimitiveSchema(Primitive("com.example.Person")),
				}),
			},
		},
		{
			name: "nullable schema type",
			src:  `schema string?;`,
			expected: []Schema{
				UnionSchema(Union{
					PrimitiveSchema(Null),
					PrimitiveSchema(String),
				}),
			},
		},
		{
			name: "record with no namespace",
			src: `schema int;
record Person {
	string name;
}`,
			expected: []Schema{
				PrimitiveSchema(Int),
				RecordSchema(Record{
					Name: "Person",
					Fields: []Field{
						{Name: "name", Type: PrimitiveSchema(String)},
					},
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, err := idl.Parse(strings.NewReader(tc.src))
			require.NoError(t, err)

			result, err := SchemaFrom(f.Schema)

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
