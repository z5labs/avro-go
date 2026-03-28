// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrinter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    *File
		expected string
	}{
		{
			name: "primitive schema with default namespace",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
				},
			},
			expected: `schema int;`,
		},
		{
			name: "primitive schema with custom namespace",
			input: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type:      Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
				},
			},
			expected: `namespace com.example;
schema int;`,
		},
		{
			name: "primitive schema with single line comment",
			input: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "// This is a comment"},
				},
				Schema: &Schema{
					Pos:  Pos{Line: 2, Column: 1},
					Type: Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
				},
			},
			expected: `// This is a comment
schema int;`,
		},
		{
			name: "primitive schema with multi line comment",
			input: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "/* This is a comment */"},
				},
				Schema: &Schema{
					Pos:  Pos{Line: 2, Column: 1},
					Type: Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
				},
			},
			expected: `/* This is a comment */
schema int;`,
		},
		{
			name: "primitive schema with namespace and comment between",
			input: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 2, Column: 1}, Text: "// comment"},
				},
				Schema: &Schema{
					Pos:       Pos{Line: 3, Column: 1},
					Namespace: "com.example",
					Type:      Ident{Pos: Pos{Line: 3, Column: 8}, Value: "int"},
				},
			},
			expected: `namespace com.example;
// comment
schema int;`,
		},
		{
			name: "primitive schema with multi multi line comment",
			input: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "/*\n* This is a comment\n*/"},
				},
				Schema: &Schema{
					Pos:  Pos{Line: 4, Column: 1},
					Type: Ident{Pos: Pos{Line: 4, Column: 8}, Value: "int"},
				},
			},
			expected: `/*
* This is a comment
*/
schema int;`,
		},
		{
			name: "comment before namespace declaration",
			input: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "// header comment"},
				},
				Schema: &Schema{
					Pos:       Pos{Line: 3, Column: 1},
					Namespace: "com.example",
					Type:      Ident{Pos: Pos{Line: 3, Column: 8}, Value: "int"},
				},
			},
			expected: `namespace com.example;
// header comment
schema int;`,
		},
		{
			name: "basic enum with values",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Value: "HEARTS"},
								{Value: "DIAMONDS"},
								{Value: "CLUBS"},
								{Value: "SPADES"},
							},
						},
					},
				},
			},
			expected: `schema int;
enum Suit {
  HEARTS,
  DIAMONDS,
  CLUBS,
  SPADES
}
`,
		},
		{
			name: "enum with default",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Value: "HEARTS"},
								{Value: "DIAMONDS"},
							},
							Default: &Ident{Value: "HEARTS"},
						},
					},
				},
			},
			expected: `schema int;
enum Suit {
  HEARTS,
  DIAMONDS
} = HEARTS;
`,
		},
		{
			name: "enum with doc comment",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Doc:  "Card suits",
							Name: "Suit",
							Values: []*Ident{
								{Value: "HEARTS"},
								{Value: "DIAMONDS"},
							},
						},
					},
				},
			},
			expected: `schema int;
/** Card suits */
enum Suit {
  HEARTS,
  DIAMONDS
}
`,
		},
		{
			name: "enum with namespace annotation",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name:      "Suit",
							Namespace: "com.example.cards",
							Values: []*Ident{
								{Value: "HEARTS"},
							},
						},
					},
				},
			},
			expected: `schema int;
@namespace("com.example.cards")
enum Suit {
  HEARTS
}
`,
		},
		{
			name: "enum with aliases annotation",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name:    "Suit",
							Aliases: []string{"OldSuit", "AncientSuit"},
							Values: []*Ident{
								{Value: "HEARTS"},
							},
						},
					},
				},
			},
			expected: `schema int;
@aliases(["OldSuit", "AncientSuit"])
enum Suit {
  HEARTS
}
`,
		},
		{
			name: "enum with custom property",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Properties: map[string]Value{
								"custom": StringValue("value"),
							},
							Values: []*Ident{
								{Value: "HEARTS"},
							},
						},
					},
				},
			},
			expected: `schema int;
@custom("value")
enum Suit {
  HEARTS
}
`,
		},
		{
			name: "enum with all annotations",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Doc:       "Card suits",
							Name:      "Suit",
							Namespace: "com.example",
							Aliases:   []string{"OldSuit"},
							Values: []*Ident{
								{Value: "HEARTS"},
								{Value: "DIAMONDS"},
							},
							Default: &Ident{Value: "HEARTS"},
						},
					},
				},
			},
			expected: `schema int;
/** Card suits */
@namespace("com.example")
@aliases(["OldSuit"])
enum Suit {
  HEARTS,
  DIAMONDS
} = HEARTS;
`,
		},
		{
			name: "nullable union schema shorthand",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							Ident{Value: "string"},
						},
					},
				},
			},
			expected: `schema string?;`,
		},
		{
			name: "nullable union schema with namespace",
			input: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							Ident{Value: "int"},
						},
					},
				},
			},
			expected: `namespace com.example;
schema int?;`,
		},
		{
			name: "multi-type union schema verbose",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							Ident{Value: "string"},
							Ident{Value: "int"},
						},
					},
				},
			},
			expected: `schema union { null, string, int };`,
		},
		{
			name: "single-type union schema verbose",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "int"},
						},
					},
				},
			},
			expected: `schema union { int };`,
		},
		{
			name: "basic map schema",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: &Map{Values: &Ident{Value: "string"}},
				},
			},
			expected: `schema map<string>;`,
		},
		{
			name: "map schema with namespace",
			input: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type:      &Map{Values: &Ident{Value: "int"}},
				},
			},
			expected: `namespace com.example;
schema map<int>;`,
		},
		{
			name: "nullable map shorthand",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							&Map{Values: &Ident{Value: "string"}},
						},
					},
				},
			},
			expected: `schema map<string>?;`,
		},
		{
			name: "map in verbose union",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							&Map{Values: &Ident{Value: "int"}},
							Ident{Value: "string"},
						},
					},
				},
			},
			expected: `schema union { null, map<int>, string };`,
		},
		{
			name: "basic array schema",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: &Array{Items: Ident{Value: "int"}},
				},
			},
			expected: `schema array<int>;`,
		},
		{
			name: "array schema with namespace",
			input: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type:      &Array{Items: Ident{Value: "string"}},
				},
			},
			expected: `namespace com.example;
schema array<string>;`,
		},
		{
			name: "nullable array shorthand",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							&Array{Items: Ident{Value: "int"}},
						},
					},
				},
			},
			expected: `schema array<int>?;`,
		},
		{
			name: "array in verbose union",
			input: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							&Array{Items: Ident{Value: "string"}},
							Ident{Value: "int"},
						},
					},
				},
			},
			expected: `schema union { null, array<string>, int };`,
		},
		{
			name: "nested map in array",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: &Array{Items: &Map{Values: &Ident{Value: "string"}}},
				},
			},
			expected: `schema array<map<string>>;`,
		},
		{
			name: "nested arrays",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: &Array{Items: &Array{Items: Ident{Value: "int"}}},
				},
			},
			expected: `schema array<array<int>>;`,
		},
		// Fixed type tests
		{
			name: "basic fixed type",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Name: "MD5",
							Size: 16,
						},
					},
				},
			},
			expected: `schema int;
fixed MD5(16);
`,
		},
		{
			name: "fixed with doc comment",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Doc:  "MD5 hash.",
							Name: "MD5",
							Size: 16,
						},
					},
				},
			},
			expected: `schema int;
/** MD5 hash. */
fixed MD5(16);
`,
		},
		{
			name: "fixed with namespace",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Name:      "MD5",
							Namespace: "org.example",
							Size:      16,
						},
					},
				},
			},
			expected: `schema int;
@namespace("org.example")
fixed MD5(16);
`,
		},
		{
			name: "fixed with aliases",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Name:    "MD5",
							Aliases: []string{"Hash", "Digest"},
							Size:    16,
						},
					},
				},
			},
			expected: `schema int;
@aliases(["Hash", "Digest"])
fixed MD5(16);
`,
		},
		{
			name: "fixed with custom property",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Name: "MD5",
							Size: 16,
							Properties: map[string]Value{
								"version": IntValue(1),
							},
						},
					},
				},
			},
			expected: `schema int;
@version(1)
fixed MD5(16);
`,
		},
		{
			name: "fixed with all annotations",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Doc:       "MD5 hash.",
							Name:      "MD5",
							Namespace: "org.example",
							Aliases:   []string{"Hash"},
							Size:      16,
							Properties: map[string]Value{
								"version": IntValue(1),
							},
						},
					},
				},
			},
			expected: `schema int;
/** MD5 hash. */
@namespace("org.example")
@aliases(["Hash"])
@version(1)
fixed MD5(16);
`,
		},
		{
			name: "multiple fixed types",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Name: "MD5",
							Size: 16,
						},
						&Fixed{
							Name: "SHA256",
							Size: 32,
						},
					},
				},
			},
			expected: `schema int;
fixed MD5(16);
fixed SHA256(32);
`,
		},
		{
			name: "basic record type",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}},
								{Name: "age", Type: Ident{Value: "int"}},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  string name;
  int age;
}
`,
		},
		{
			name: "record with doc comment",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Doc:  "A person.",
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
/** A person. */
record Person {
  string name;
}
`,
		},
		{
			name: "record with namespace",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name:      "Person",
							Namespace: "com.example",
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
@namespace("com.example")
record Person {
  string name;
}
`,
		},
		{
			name: "record with aliases",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name:    "Person",
							Aliases: []string{"Human", "Individual"},
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
@aliases(["Human", "Individual"])
record Person {
  string name;
}
`,
		},
		{
			name: "record with properties",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Properties: map[string]Value{
								"version": IntValue(2),
							},
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
@version(2)
record Person {
  string name;
}
`,
		},
		{
			name: "record with all annotations",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Doc:       "A person.",
							Name:      "Person",
							Namespace: "com.example",
							Aliases:   []string{"Human"},
							Properties: map[string]Value{
								"version": IntValue(1),
							},
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
/** A person. */
@namespace("com.example")
@aliases(["Human"])
@version(1)
record Person {
  string name;
}
`,
		},
		{
			name: "record field with doc comment",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Doc: "The name.", Name: "name", Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  /** The name. */
  string name;
}
`,
		},
		{
			name: "record field with aliases",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Aliases: []string{"fullName"}, Type: Ident{Value: "string"}},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  @aliases(["fullName"])
  string name;
}
`,
		},
		{
			name: "record field with order descending",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}, SortOrder: SortOrderDesc},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  string @order("descending") name;
}
`,
		},
		{
			name: "record field with order ignore",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}, SortOrder: SortOrderIgnore},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  string @order("ignore") name;
}
`,
		},
		{
			name: "record field with default string",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Type: Ident{Value: "string"}, Default: StringValue("unknown")},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  string name = "unknown";
}
`,
		},
		{
			name: "record field with default int",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "age", Type: Ident{Value: "int"}, Default: IntValue(0)},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  int age = 0;
}
`,
		},
		{
			name: "record field with default null",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "name", Type: &Union{Types: []Type{Ident{Value: "null"}, Ident{Value: "string"}}}, Default: NullValue{}},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  string? name = null;
}
`,
		},
		{
			name: "record field with all annotations",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{
									Doc:       "The name.",
									Name:      "name",
									Aliases:   []string{"fullName"},
									Type:      Ident{Value: "string"},
									SortOrder: SortOrderDesc,
									Default:   StringValue("unknown"),
								},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  /** The name. */
  @aliases(["fullName"])
  string @order("descending") name = "unknown";
}
`,
		},
		{
			name: "record with nested types",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Person",
							Fields: []*Field{
								{Name: "tags", Type: &Array{Items: Ident{Value: "string"}}},
								{Name: "metadata", Type: &Map{Values: &Ident{Value: "string"}}},
							},
						},
					},
				},
			},
			expected: `schema int;
record Person {
  array<string> tags;
  map<string> metadata;
}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := Print(&buf, tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestPrinterRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
	}{
		{
			name: "primitive schema with default namespace",
			src:  `schema int;`,
		},
		{
			name: "primitive schema with custom namespace",
			src: `namespace com.example;
schema int;`,
		},
		{
			name: "primitive schema with single line comment",
			src: `// This is a comment
schema int;`,
		},
		{
			name: "primitive schema with multi line comment",
			src: `/* This is a comment */
schema int;`,
		},
		{
			name: "primitive schema with namespace and comment between",
			src: `namespace com.example;
// comment
schema int;`,
		},
		{
			name: "primitive schema with multi multi line comment",
			src: `/*
* This is a comment
*/
schema int;`,
		},
		{
			name: "comment before namespace declaration",
			src: `// header comment
namespace com.example;
schema int;`,
		},
		{
			name: "basic enum",
			src: `schema int;
enum Suit {
  HEARTS,
  DIAMONDS,
  CLUBS,
  SPADES
}
`,
		},
		{
			name: "enum with default",
			src: `schema int;
enum Suit {
  HEARTS,
  DIAMONDS
} = HEARTS;
`,
		},
		{
			name: "enum with all annotations",
			src: `schema int;
/** Card suits */
@namespace("com.example")
@aliases(["OldSuit", "AncientSuit"])
enum Suit {
  HEARTS,
  DIAMONDS
} = HEARTS;
`,
		},
		{
			name: "nullable union shorthand",
			src:  `schema string?;`,
		},
		{
			name: "nullable union with namespace",
			src: `namespace com.example;
schema int?;`,
		},
		{
			name: "multi-type union",
			src:  `schema union { null, string, int };`,
		},
		{
			name: "single-type union",
			src:  `schema union { int };`,
		},
		{
			name: "basic map schema",
			src:  `schema map<string>;`,
		},
		{
			name: "map schema with namespace",
			src: `namespace com.example;
schema map<int>;`,
		},
		{
			name: "basic fixed type",
			src: `schema int;
fixed MD5(16);
`,
		},
		{
			name: "fixed with namespace",
			src: `schema int;
@namespace("org.example")
fixed MD5(16);
`,
		},
		{
			name: "fixed with all annotations",
			src: `schema int;
/** MD5 hash. */
@namespace("org.example")
@aliases(["Hash", "Digest"])
fixed MD5(16);
`,
		},
		{
			name: "basic record",
			src: `schema int;
record Person {
  string name;
  int age;
}
`,
		},
		{
			name: "record with namespace",
			src: `schema int;
@namespace("com.example")
record Person {
  string name;
}
`,
		},
		{
			name: "record with all annotations",
			src: `schema int;
/** A person. */
@namespace("com.example")
@aliases(["Human"])
record Person {
  string name;
}
`,
		},
		{
			name: "record field with doc",
			src: `schema int;
record Person {
  /** The name. */
  string name;
}
`,
		},
		{
			name: "record field with aliases",
			src: `schema int;
record Person {
  @aliases(["fullName"])
  string name;
}
`,
		},
		{
			name: "record field with order descending",
			src: `schema int;
record Person {
  string @order("descending") name;
}
`,
		},
		{
			name: "record field with default",
			src: `schema int;
record Person {
  string name = "unknown";
}
`,
		},
		{
			name: "record field with all annotations",
			src: `schema int;
record Person {
  /** The name. */
  @aliases(["fullName"])
  string @order("descending") name = "unknown";
}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Parse the original source
			file1, err := Parse(strings.NewReader(tc.src))
			require.NoError(t, err)

			// Print the parsed AST
			var buf bytes.Buffer
			err = Print(&buf, file1)
			require.NoError(t, err)

			// Parse the printed output
			file2, err := Parse(strings.NewReader(buf.String()))
			require.NoError(t, err)

			// Compare ASTs (ignoring position info)
			require.Equal(t, file1.Schema.Namespace, file2.Schema.Namespace)
			require.Equal(t, len(file1.Comments), len(file2.Comments))
			for i := range file1.Comments {
				require.Equal(t, file1.Comments[i].Text, file2.Comments[i].Text)
			}

			// Compare schema types
			switch t1 := file1.Schema.Type.(type) {
			case Ident:
				t2, ok := file2.Schema.Type.(Ident)
				require.True(t, ok)
				require.Equal(t, t1.Value, t2.Value)
			case *Union:
				t2, ok := file2.Schema.Type.(*Union)
				require.True(t, ok, "expected *Union")
				require.Equal(t, len(t1.Types), len(t2.Types))
				for j := range t1.Types {
					id1, ok1 := t1.Types[j].(Ident)
					id2, ok2 := t2.Types[j].(Ident)
					require.True(t, ok1 && ok2, "expected Ident types in union")
					require.Equal(t, id1.Value, id2.Value)
				}
			case *Map:
				t2, ok := file2.Schema.Type.(*Map)
				require.True(t, ok, "expected *Map")
				require.Equal(t, t1.Values.Value, t2.Values.Value)
			}

			// Compare Schema.Types (e.g., enums, records)
			require.Equal(t, len(file1.Schema.Types), len(file2.Schema.Types))
			for i := range file1.Schema.Types {
				switch typ1 := file1.Schema.Types[i].(type) {
				case *Enum:
					typ2, ok := file2.Schema.Types[i].(*Enum)
					require.True(t, ok, "expected *Enum at index %d", i)
					require.Equal(t, typ1.Doc, typ2.Doc)
					require.Equal(t, typ1.Name, typ2.Name)
					require.Equal(t, typ1.Namespace, typ2.Namespace)
					require.Equal(t, typ1.Aliases, typ2.Aliases)
					require.Equal(t, len(typ1.Values), len(typ2.Values))
					for j := range typ1.Values {
						require.Equal(t, typ1.Values[j].Value, typ2.Values[j].Value)
					}
					if typ1.Default != nil {
						require.NotNil(t, typ2.Default)
						require.Equal(t, typ1.Default.Value, typ2.Default.Value)
					} else {
						require.Nil(t, typ2.Default)
					}
					require.Equal(t, len(typ1.Properties), len(typ2.Properties))
					for k, v := range typ1.Properties {
						require.Equal(t, v, typ2.Properties[k])
					}
				case *Fixed:
					typ2, ok := file2.Schema.Types[i].(*Fixed)
					require.True(t, ok, "expected *Fixed at index %d", i)
					require.Equal(t, typ1.Doc, typ2.Doc)
					require.Equal(t, typ1.Name, typ2.Name)
					require.Equal(t, typ1.Namespace, typ2.Namespace)
					require.Equal(t, typ1.Aliases, typ2.Aliases)
					require.Equal(t, typ1.Size, typ2.Size)
					require.Equal(t, len(typ1.Properties), len(typ2.Properties))
					for k, v := range typ1.Properties {
						require.Equal(t, v, typ2.Properties[k])
					}
				case *Record:
					typ2, ok := file2.Schema.Types[i].(*Record)
					require.True(t, ok, "expected *Record at index %d", i)
					require.Equal(t, typ1.Doc, typ2.Doc)
					require.Equal(t, typ1.Name, typ2.Name)
					require.Equal(t, typ1.Namespace, typ2.Namespace)
					require.Equal(t, typ1.Aliases, typ2.Aliases)
					require.Equal(t, len(typ1.Properties), len(typ2.Properties))
					for k, v := range typ1.Properties {
						require.Equal(t, v, typ2.Properties[k])
					}
					require.Equal(t, len(typ1.Fields), len(typ2.Fields))
					for j := range typ1.Fields {
						f1, f2 := typ1.Fields[j], typ2.Fields[j]
						require.Equal(t, f1.Doc, f2.Doc)
						require.Equal(t, f1.Name, f2.Name)
						require.Equal(t, f1.Aliases, f2.Aliases)
						require.Equal(t, f1.SortOrder, f2.SortOrder)
						require.Equal(t, f1.Default, f2.Default)
					}
				}
			}
		})
	}
}

func TestPrinterErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       *File
		expectedErr string
	}{
		{
			name: "empty union",
			input: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: &Union{Types: []Type{}},
				},
			},
			expectedErr: "idl: union must have at least one type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := Print(&buf, tc.input)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
