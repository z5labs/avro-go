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
enum Suit { HEARTS, DIAMONDS, CLUBS, SPADES }
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
enum Suit { HEARTS, DIAMONDS } = HEARTS;
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
enum Suit { HEARTS, DIAMONDS }
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
enum Suit { HEARTS }
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
enum Suit { HEARTS }
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
enum Suit { HEARTS }
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
enum Suit { HEARTS, DIAMONDS } = HEARTS;
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
enum Suit { HEARTS, DIAMONDS, CLUBS, SPADES }
`,
		},
		{
			name: "enum with default",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS } = HEARTS;
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
			}

			// Compare Schema.Types (e.g., enums, records)
			require.Equal(t, len(file1.Schema.Types), len(file2.Schema.Types))
			for i := range file1.Schema.Types {
				switch typ1 := file1.Schema.Types[i].(type) {
				case *Enum:
					typ2, ok := file2.Schema.Types[i].(*Enum)
					require.True(t, ok, "expected *Enum at index %d", i)
					require.Equal(t, typ1.Name, typ2.Name)
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
				}
			}
		})
	}
}
