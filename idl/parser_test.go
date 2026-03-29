// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParserErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		src            string
		expectedErrMsg string
		expectedErr    error
	}{
		{
			name:        "empty input",
			src:         ``,
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenComment, TokenDocComment}},
		},
		{
			name:           "file starts with unrecognized keyword",
			src:            `invalid int;`,
			expectedErrMsg: "schema idl must start with either 'schema' or 'namespace'",
		},
		{
			name: "namespace not followed by schema keyword",
			src: `namespace com.example;
invalid int;`,
			expectedErrMsg: "schema definition must follow namespace declaration",
		},
		{
			name: "schema missing type identifier",
			src:  `schema ;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 1, Column: 8}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name:        "schema type missing semicolon",
			src:         "schema int ",
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenSymbol}},
		},
		{
			name: "schema type followed by wrong symbol",
			src:  `schema int }`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 1, Column: 12}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "enum missing name",
			src: `schema int;
enum { HEARTS };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 2, Column: 6}, Type: TokenSymbol, Value: []byte("{")},
			},
		},
		{
			name: "enum missing open brace",
			src: `schema int;
enum Suit HEARTS };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 2, Column: 11}, Type: TokenIdentifier, Value: []byte("HEARTS")},
			},
		},
		{
			name: "enum missing close brace",
			src: `schema int;
enum Suit { HEARTS ;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 2, Column: 20}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "enum empty body",
			src: `schema int;
enum Suit { };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 2, Column: 13}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "enum default missing identifier",
			src: `schema int;
enum Suit { HEARTS } =`,
			expectedErr: UnexpectedEndOfTokensError{
				Expected: []TokenType{TokenIdentifier},
			},
		},
		{
			name:        "namespace missing semicolon",
			src:         "namespace com.example ",
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenSymbol}},
		},
		{
			name:        "namespace with no following tokens",
			src:         `namespace com.example;`,
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenComment, TokenDocComment}},
		},
		{
			name: "fixed missing name",
			src: `schema int;
fixed (16);`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 2, Column: 7}, Type: TokenSymbol, Value: []byte("(")},
			},
		},
		{
			name: "fixed missing open paren",
			src: `schema int;
fixed MD5 16);`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 2, Column: 11}, Type: TokenNumber, Value: []byte("16")},
			},
		},
		{
			name: "fixed missing size",
			src: `schema int;
fixed MD5();`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenNumber},
				Actual:   Token{Pos: Pos{Line: 2, Column: 11}, Type: TokenSymbol, Value: []byte(")")},
			},
		},
		{
			name: "fixed missing close paren",
			src: `schema int;
fixed MD5(16;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 2, Column: 13}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "map schema missing open angle bracket",
			src:  `schema map int>;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 1, Column: 12}, Type: TokenIdentifier, Value: []byte("int")},
			},
		},
		{
			name: "map schema missing value type",
			src:  `schema map<>;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 1, Column: 12}, Type: TokenSymbol, Value: []byte(">")},
			},
		},
		{
			name: "map schema missing close angle bracket",
			src:  `schema map<int;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 1, Column: 15}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "union schema missing open brace",
			src:  `schema union null, string };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 1, Column: 14}, Type: TokenIdentifier, Value: []byte("null")},
			},
		},
		{
			name: "union schema empty body",
			src:  `schema union { };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 1, Column: 16}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "union schema missing close brace",
			src:  `schema union { null, string ;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 1, Column: 29}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "record missing name",
			src: `schema int;
record { string name; };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 2, Column: 8}, Type: TokenSymbol, Value: []byte("{")},
			},
		},
		{
			name: "record missing open brace",
			src: `schema int;
record Employee string name; };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 2, Column: 17}, Type: TokenIdentifier, Value: []byte("string")},
			},
		},
		{
			name: "record empty body",
			src: `schema int;
record Employee { };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 2, Column: 19}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "record field missing name",
			src: `schema int;
record Employee { string ; };`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier, TokenSymbol, TokenAnnotation},
				Actual:   Token{Pos: Pos{Line: 2, Column: 26}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "record field missing semicolon",
			src: `schema int;
record Employee {
  string name
}`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 4, Column: 1}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "record missing close brace",
			src: `schema int;
record Employee {
  string name;
;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier, TokenSymbol},
				Actual:   Token{Pos: Pos{Line: 4, Column: 1}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(strings.NewReader(tc.src))

			if tc.expectedErrMsg != "" {
				require.EqualError(t, err, tc.expectedErrMsg)
				return
			}
			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestParser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		src      string
		expected *File
	}{
		{
			name: "primitive schema with default namespace",
			src:  `schema int;`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
				},
			},
		},
		{
			name: "primitive schema with custom namespace",
			src: `namespace com.example;
schema int;`,
			expected: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type:      Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
				},
			},
		},
		{
			name: "primitive schema with single line comment",
			src: `// This is a comment
schema int;`,
			expected: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "// This is a comment"},
				},
				Schema: &Schema{
					Pos:  Pos{Line: 2, Column: 1},
					Type: Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
				},
			},
		},
		{
			name: "primitive schema with multi single line comment",
			src: `/* This is a comment */
schema int;`,
			expected: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "/* This is a comment */"},
				},
				Schema: &Schema{
					Pos:  Pos{Line: 2, Column: 1},
					Type: Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
				},
			},
		},
		{
			name: "primitive schema with namespace and comment between",
			src: `namespace com.example;
// comment
schema int;`,
			expected: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 2, Column: 1}, Text: "// comment"},
				},
				Schema: &Schema{
					Pos:       Pos{Line: 3, Column: 1},
					Namespace: "com.example",
					Type:      Ident{Pos: Pos{Line: 3, Column: 8}, Value: "int"},
				},
			},
		},
		{
			name: "primitive schema with multi multi line comment",
			src: `/*
* This is a comment
*/
schema int;`,
			expected: &File{
				Comments: []*Comment{
					{Pos: Pos{Line: 1, Column: 1}, Text: "/*\n* This is a comment\n*/"},
				},
				Schema: &Schema{
					Pos:  Pos{Line: 4, Column: 1},
					Type: Ident{Pos: Pos{Line: 4, Column: 8}, Value: "int"},
				},
			},
		},
		{
			name: "schema with single enum type",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS, CLUBS, SPADES }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 2, Column: 21}, Value: "DIAMONDS"},
								{Pos: Pos{Line: 2, Column: 31}, Value: "CLUBS"},
								{Pos: Pos{Line: 2, Column: 38}, Value: "SPADES"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type and default",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS } = HEARTS;`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 2, Column: 21}, Value: "DIAMONDS"},
							},
							Default: &Ident{Pos: Pos{Line: 2, Column: 34}, Value: "HEARTS"},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type and default followed by another type",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS } = HEARTS;
enum Color { RED, BLACK }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 2, Column: 21}, Value: "DIAMONDS"},
							},
							Default: &Ident{Pos: Pos{Line: 2, Column: 34}, Value: "HEARTS"},
						},
						&Enum{
							Name: "Color",
							Values: []*Ident{
								{Pos: Pos{Line: 3, Column: 14}, Value: "RED"},
								{Pos: Pos{Line: 3, Column: 19}, Value: "BLACK"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type trailing comma",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS, }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 2, Column: 21}, Value: "DIAMONDS"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type single value",
			src: `schema int;
enum Suit { HEARTS }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type and doc comments on values",
			src: `schema int;
enum Suit {
  /** Hearts suit */
  HEARTS,
  /** Diamonds suit */
  DIAMONDS,
  /** Clubs suit */
  CLUBS,
  /** Spades suit */
  SPADES
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 4, Column: 3}, Value: "HEARTS"},
								{Pos: Pos{Line: 6, Column: 3}, Value: "DIAMONDS"},
								{Pos: Pos{Line: 8, Column: 3}, Value: "CLUBS"},
								{Pos: Pos{Line: 10, Column: 3}, Value: "SPADES"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type and line comments on values",
			src: `schema int;
enum Suit {
  // Hearts suit
  HEARTS,
  // Diamonds suit
  DIAMONDS
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 4, Column: 3}, Value: "HEARTS"},
								{Pos: Pos{Line: 6, Column: 3}, Value: "DIAMONDS"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum type and doc comment before first value",
			src: `schema int;
enum Suit {
  /** First value */
  HEARTS
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 4, Column: 3}, Value: "HEARTS"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with multiple enum types",
			src: `schema int;
enum Suit { HEARTS }
enum Color { RED, BLACK }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
							},
						},
						&Enum{
							Name: "Color",
							Values: []*Ident{
								{Pos: Pos{Line: 3, Column: 14}, Value: "RED"},
								{Pos: Pos{Line: 3, Column: 19}, Value: "BLACK"},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with single fixed type",
			src: `schema int;
fixed MD5(16);`,
			expected: &File{
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
		},
		{
			name: "schema with multiple fixed types",
			src: `schema int;
fixed MD5(16);
fixed SHA256(32);`,
			expected: &File{
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
		},
		{
			name: "schema with enum and fixed types",
			src: `schema int;
enum Suit { HEARTS }
fixed MD5(16);`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
							},
						},
						&Fixed{
							Name: "MD5",
							Size: 16,
						},
					},
				},
			},
		},
		{
			name: "map schema with default namespace",
			src:  `schema map<int>;`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Map{
						Values: &Ident{Pos: Pos{Line: 1, Column: 12}, Value: "int"},
					},
				},
			},
		},
		{
			name: "map schema with custom namespace",
			src: `namespace com.example;
schema map<string>;`,
			expected: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type: &Map{
						Values: &Ident{Pos: Pos{Line: 2, Column: 12}, Value: "string"},
					},
				},
			},
		},
		{
			name: "map schema with enum type",
			src: `schema map<Suit>;
enum Suit { HEARTS, DIAMONDS }`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Map{
						Values: &Ident{Pos: Pos{Line: 1, Column: 12}, Value: "Suit"},
					},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 2, Column: 21}, Value: "DIAMONDS"},
							},
						},
					},
				},
			},
		},
		{
			name: "union schema with primitives",
			src:  `schema union { null, string };`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Pos: Pos{Line: 1, Column: 16}, Value: "null"},
							Ident{Pos: Pos{Line: 1, Column: 22}, Value: "string"},
						},
					},
				},
			},
		},
		{
			name: "union schema single type",
			src:  `schema union { int };`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Pos: Pos{Line: 1, Column: 16}, Value: "int"},
						},
					},
				},
			},
		},
		{
			name: "union schema with trailing comma",
			src:  `schema union { null, string, };`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Pos: Pos{Line: 1, Column: 16}, Value: "null"},
							Ident{Pos: Pos{Line: 1, Column: 22}, Value: "string"},
						},
					},
				},
			},
		},
		{
			name: "union schema with namespace",
			src: `namespace com.example;
schema union { null, int };`,
			expected: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type: &Union{
						Types: []Type{
							Ident{Pos: Pos{Line: 2, Column: 16}, Value: "null"},
							Ident{Pos: Pos{Line: 2, Column: 22}, Value: "int"},
						},
					},
				},
			},
		},
		{
			name: "union schema with enum type",
			src: `schema union { null, Suit };
enum Suit { HEARTS, DIAMONDS }`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Pos: Pos{Line: 1, Column: 16}, Value: "null"},
							Ident{Pos: Pos{Line: 1, Column: 22}, Value: "Suit"},
						},
					},
					Types: []Type{
						&Enum{
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 2, Column: 21}, Value: "DIAMONDS"},
							},
						},
					},
				},
			},
		},
		{
			name: "nullable shorthand schema",
			src:  `schema string?;`,
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							Ident{Pos: Pos{Line: 1, Column: 8}, Value: "string"},
						},
					},
				},
			},
		},
		{
			name: "nullable shorthand schema with namespace",
			src: `namespace com.example;
schema int?;`,
			expected: &File{
				Schema: &Schema{
					Pos:       Pos{Line: 2, Column: 1},
					Namespace: "com.example",
					Type: &Union{
						Types: []Type{
							Ident{Value: "null"},
							Ident{Pos: Pos{Line: 2, Column: 8}, Value: "int"},
						},
					},
				},
			},
		},
		{
			name: "schema with single record type with single field",
			src: `schema int;
record Employee {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with single record type with multiple fields",
			src: `schema int;
record Employee {
  string name;
  boolean active;
  long salary;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
								},
								{
									Name: "active",
									Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "boolean"},
								},
								{
									Name: "salary",
									Type: Ident{Pos: Pos{Line: 5, Column: 3}, Value: "long"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with record type with nullable field",
			src: `schema int;
record Employee {
  string? name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: &Union{
										Types: []Type{
											Ident{Value: "null"},
											Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with record type with map field",
			src: `schema int;
record Employee {
  map<string> metadata;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "metadata",
									Type: &Map{
										Values: &Ident{Pos: Pos{Line: 3, Column: 7}, Value: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with record type with union field",
			src: `schema int;
record Employee {
  union { null, string } name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: &Union{
										Types: []Type{
											Ident{Pos: Pos{Line: 3, Column: 11}, Value: "null"},
											Ident{Pos: Pos{Line: 3, Column: 17}, Value: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with multiple record types",
			src: `schema int;
record Employee {
  string name;
}
record Department {
  string title;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
								},
							},
						},
						&Record{
							Name: "Department",
							Fields: []*Field{
								{
									Name: "title",
									Type: Ident{Pos: Pos{Line: 6, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with record followed by enum",
			src: `schema int;
record Employee {
  string name;
}
enum Status { ACTIVE, INACTIVE }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
								},
							},
						},
						&Enum{
							Name: "Status",
							Values: []*Ident{
								{Pos: Pos{Line: 5, Column: 15}, Value: "ACTIVE"},
								{Pos: Pos{Line: 5, Column: 23}, Value: "INACTIVE"},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with null default value",
			src: `schema int;
record Config {
  int? value = null;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "value",
									Type:    &Union{Types: []Type{Ident{Value: "null"}, Ident{Pos: Pos{Line: 3, Column: 3}, Value: "int"}}},
									Default: NullValue{},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with boolean default value",
			src: `schema int;
record Config {
  boolean active = true;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "active",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "boolean"},
									Default: BoolValue(true),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with integer default value",
			src: `schema int;
record Config {
  int count = 42;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "count",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "int"},
									Default: IntValue(42),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with float default value",
			src: `schema int;
record Config {
  double rate = 3.14;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "rate",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "double"},
									Default: FloatValue(3.14),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with negative number default value",
			src: `schema int;
record Config {
  int offset = -1;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "offset",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "int"},
									Default: IntValue(-1),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with string default value",
			src: `schema int;
record Config {
  string name = "hello";
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "name",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
									Default: StringValue("hello"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with empty string default value",
			src: `schema int;
record Config {
  string name = "";
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "name",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
									Default: StringValue(""),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with array default value",
			src: `schema int;
record Config {
  Nums nums = [1, 2, 3];
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "nums",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "Nums"},
									Default: ArrayValue{IntValue(1), IntValue(2), IntValue(3)},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with empty array default value",
			src: `schema int;
record Config {
  Nums nums = [];
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name:    "nums",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "Nums"},
									Default: ArrayValue(nil),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with object default value",
			src: `schema int;
record Config {
  map<string> meta = {"key": "val"};
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name: "meta",
									Type: &Map{
										Values: &Ident{Pos: Pos{Line: 3, Column: 7}, Value: "string"},
									},
									Default: ObjectValue{"key": StringValue("val")},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with empty object default value",
			src: `schema int;
record Config {
  map<string> meta = {};
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name: "meta",
									Type: &Map{
										Values: &Ident{Pos: Pos{Line: 3, Column: 7}, Value: "string"},
									},
									Default: ObjectValue{},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record field with nested array default value",
			src: `schema int;
record Config {
  Grid grid = [[1, 2], [3, 4]];
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Config",
							Fields: []*Field{
								{
									Name: "grid",
									Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "Grid"},
									Default: ArrayValue{
										ArrayValue{IntValue(1), IntValue(2)},
										ArrayValue{IntValue(3), IntValue(4)},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record with mixed fields with and without defaults",
			src: `schema int;
record Employee {
  string name;
  boolean active = true;
  int age;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
								},
								{
									Name:    "active",
									Type:    Ident{Pos: Pos{Line: 4, Column: 3}, Value: "boolean"},
									Default: BoolValue(true),
								},
								{
									Name: "age",
									Type: Ident{Pos: Pos{Line: 5, Column: 3}, Value: "int"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "schema with enum followed by record",
			src: `schema int;
enum Status { ACTIVE, INACTIVE }
record Employee {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Status",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 15}, Value: "ACTIVE"},
								{Pos: Pos{Line: 2, Column: 23}, Value: "INACTIVE"},
							},
						},
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record with namespace annotation",
			src: `schema int;
@namespace("org.example")
record Employee {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name:      "Employee",
							Namespace: "org.example",
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record with aliases annotation",
			src: `schema int;
@aliases(["org.old.OldRecord", "org.ancient.AncientRecord"])
record MyRecord {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name:    "MyRecord",
							Aliases: []string{"org.old.OldRecord", "org.ancient.AncientRecord"},
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record with namespace and aliases annotations",
			src: `schema int;
@namespace("org.example")
@aliases(["org.old.OldRecord"])
record MyRecord {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name:      "MyRecord",
							Namespace: "org.example",
							Aliases:   []string{"org.old.OldRecord"},
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 5, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "record with custom annotation",
			src: `schema int;
@custom("val")
record Foo {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name:       "Foo",
							Properties: map[string]Value{"custom": StringValue("val")},
							Fields: []*Field{
								{
									Name: "name",
									Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "string"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "field with order annotation ascending",
			src: `schema int;
record MyRecord {
  string @order("ascending") name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "MyRecord",
							Fields: []*Field{
								{
									Name:      "name",
									Type:      Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
									SortOrder: SortOrderAsc,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "field with order annotation descending",
			src: `schema int;
record MyRecord {
  string @order("descending") name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "MyRecord",
							Fields: []*Field{
								{
									Name:      "name",
									Type:      Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
									SortOrder: SortOrderDesc,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "field with order annotation ignore",
			src: `schema int;
record MyRecord {
  string @order("ignore") name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "MyRecord",
							Fields: []*Field{
								{
									Name:      "name",
									Type:      Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
									SortOrder: SortOrderIgnore,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "field with aliases annotation",
			src: `schema int;
record MyRecord {
  string @aliases(["oldField", "ancientField"]) myNewField;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "MyRecord",
							Fields: []*Field{
								{
									Name:    "myNewField",
									Type:    Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"},
									Aliases: []string{"oldField", "ancientField"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "field with pre-type custom annotation",
			src: `schema int;
record Foo {
  @custom("val") string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Foo",
							Fields: []*Field{
								{
									Name:       "name",
									Type:       Ident{Pos: Pos{Line: 3, Column: 18}, Value: "string"},
									Properties: map[string]Value{"custom": StringValue("val")},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "enum with namespace annotation",
			src: `schema int;
@namespace("org.example")
enum Suit { HEARTS, DIAMONDS }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name:      "Suit",
							Namespace: "org.example",
							Values: []*Ident{
								{Pos: Pos{Line: 3, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 3, Column: 21}, Value: "DIAMONDS"},
							},
						},
					},
				},
			},
		},
		{
			name: "fixed with namespace annotation",
			src: `schema int;
@namespace("org.example")
fixed MD5(16);`,
			expected: &File{
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
		},
		{
			name: "doc comment before record",
			src: `schema int;
/** This is a record. */
record Employee {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Doc:  "This is a record.",
							Name: "Employee",
							Fields: []*Field{
								{Name: "name", Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "string"}},
							},
						},
					},
				},
			},
		},
		{
			name: "doc comment before field",
			src: `schema int;
record Employee {
  /** Employee's full name. */
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{Doc: "Employee's full name.", Name: "name", Type: Ident{Pos: Pos{Line: 4, Column: 3}, Value: "string"}},
							},
						},
					},
				},
			},
		},
		{
			name: "doc comment before enum",
			src: `schema int;
/** Card suits. */
enum Suit { HEARTS, DIAMONDS }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Doc:  "Card suits.",
							Name: "Suit",
							Values: []*Ident{
								{Pos: Pos{Line: 3, Column: 13}, Value: "HEARTS"},
								{Pos: Pos{Line: 3, Column: 21}, Value: "DIAMONDS"},
							},
						},
					},
				},
			},
		},
		{
			name: "doc comment before fixed",
			src: `schema int;
/** MD5 hash. */
fixed MD5(16);`,
			expected: &File{
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
		},
		{
			name: "doc comment with annotations on record",
			src: `schema int;
/** Documented record. */
@namespace("org.example")
record Employee {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Doc:       "Documented record.",
							Name:      "Employee",
							Namespace: "org.example",
							Fields: []*Field{
								{Name: "name", Type: Ident{Pos: Pos{Line: 5, Column: 3}, Value: "string"}},
							},
						},
					},
				},
			},
		},
		{
			name: "multi line doc comment stripped",
			src: `schema int;
/**
 * This is a multi-line
 * documentation string.
 */
record Employee {
  string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Doc:  "This is a multi-line\ndocumentation string.",
							Name: "Employee",
							Fields: []*Field{
								{Name: "name", Type: Ident{Pos: Pos{Line: 7, Column: 3}, Value: "string"}},
							},
						},
					},
				},
			},
		},
		{
			name: "doc comment on second field",
			src: `schema int;
record Employee {
  string name;
  /** Employee's age. */
  int age;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{Name: "name", Type: Ident{Pos: Pos{Line: 3, Column: 3}, Value: "string"}},
								{Doc: "Employee's age.", Name: "age", Type: Ident{Pos: Pos{Line: 5, Column: 3}, Value: "int"}},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(strings.NewReader(tc.src))

			require.NoError(t, err)
			require.Equal(t, tc.expected, file)
		})
	}
}

func TestParserEscapedIdentifiers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		src      string
		expected *File
	}{
		{
			name: "escaped record name",
			src: `schema int;
record ` + "`schema`" + ` {
	string name;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "schema",
							Fields: []*Field{
								{Name: "name", Type: Ident{Pos: Pos{Line: 3, Column: 2}, Value: "string"}},
							},
						},
					},
				},
			},
		},
		{
			name: "escaped enum name",
			src: `schema int;
enum ` + "`error`" + ` { VALUE }`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "error",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 16}, Value: "VALUE"},
							},
						},
					},
				},
			},
		},
		{
			name: "escaped fixed name",
			src:  "schema int;\nfixed `null`(16);",
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Fixed{
							Name: "null",
							Size: 16,
						},
					},
				},
			},
		},
		{
			name: "escaped field name",
			src: `schema int;
record Employee {
	string ` + "`namespace`" + `;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{Name: "namespace", Type: Ident{Pos: Pos{Line: 3, Column: 2}, Value: "string"}},
							},
						},
					},
				},
			},
		},
		{
			name: "escaped enum value",
			src:  "schema int;\nenum Color { `null`, RED }",
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Color",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 14}, Value: "null"},
								{Pos: Pos{Line: 2, Column: 22}, Value: "RED"},
							},
						},
					},
				},
			},
		},
		{
			name: "escaped type reference in record field",
			src: `schema int;
record Employee {
	` + "`schema`" + ` data;
}`,
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Record{
							Name: "Employee",
							Fields: []*Field{
								{Name: "data", Type: Ident{Pos: Pos{Line: 3, Column: 2}, Value: "schema"}},
							},
						},
					},
				},
			},
		},
		{
			name: "escaped type in union",
			src:  "schema union { `null`, `map` };",
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Union{
						Types: []Type{
							Ident{Pos: Pos{Line: 1, Column: 16}, Value: "null"},
							Ident{Pos: Pos{Line: 1, Column: 24}, Value: "map"},
						},
					},
				},
			},
		},
		{
			name: "escaped type in map",
			src:  "schema map<`union`>;",
			expected: &File{
				Schema: &Schema{
					Pos: Pos{Line: 1, Column: 1},
					Type: &Map{
						Values: &Ident{Pos: Pos{Line: 1, Column: 12}, Value: "union"},
					},
				},
			},
		},
		{
			name: "escaped enum default value",
			src:  "schema int;\nenum Status { `null`, ACTIVE } = `null`;",
			expected: &File{
				Schema: &Schema{
					Pos:  Pos{Line: 1, Column: 1},
					Type: Ident{Pos: Pos{Line: 1, Column: 8}, Value: "int"},
					Types: []Type{
						&Enum{
							Name: "Status",
							Values: []*Ident{
								{Pos: Pos{Line: 2, Column: 15}, Value: "null"},
								{Pos: Pos{Line: 2, Column: 23}, Value: "ACTIVE"},
							},
							Default: &Ident{Pos: Pos{Line: 2, Column: 34}, Value: "null"},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(strings.NewReader(tc.src))

			require.NoError(t, err)
			require.Equal(t, tc.expected, file)
		})
	}
}

func TestParserEscapedIdentifierErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		src         string
		expectedErr error
	}{
		{
			name:        "unterminated escaped identifier in record name",
			src:         "schema int;\nrecord `schema { string name; }",
			expectedErr: UnterminatedEscapedIdentifierError{Pos: Pos{Line: 2, Column: 8}},
		},
		{
			name:        "unterminated escaped identifier in enum value",
			src:         "schema int;\nenum Color { `null }",
			expectedErr: UnterminatedEscapedIdentifierError{Pos: Pos{Line: 2, Column: 14}},
		},
		{
			name:        "tokenizer error inside escaped identifier is preserved",
			src:         "schema int;\nrecord `name$ { }",
			expectedErr: UnexpectedCharacterError{Pos: Pos{Line: 2, Column: 14}, R: '$'},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(strings.NewReader(tc.src))

			require.Equal(t, tc.expectedErr, err)
		})
	}
}
