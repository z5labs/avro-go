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
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenComment}},
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
			name: "enum missing semicolon",
			src: `schema int;
enum Suit { HEARTS }`,
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenSymbol}},
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
enum Suit { HEARTS } = ;`,
			expectedErr: UnexpectedTokenError{
				Expected: []TokenType{TokenIdentifier},
				Actual:   Token{Pos: Pos{Line: 2, Column: 24}, Type: TokenSymbol, Value: []byte(";")},
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
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenComment}},
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
enum Suit { HEARTS, DIAMONDS, CLUBS, SPADES };`,
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
			name: "schema with enum type trailing comma",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS, };`,
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
enum Suit { HEARTS };`,
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
			name: "schema with multiple enum types",
			src: `schema int;
enum Suit { HEARTS };
enum Color { RED, BLACK };`,
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
enum Suit { HEARTS };
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
