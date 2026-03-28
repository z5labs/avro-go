// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"iter"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenizerErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		src         string
		expectedErr error
	}{
		{
			name: "unexpected character at start of input",
			src:  `$schema int;`,
			expectedErr: UnexpectedCharacterError{
				Pos: Pos{Line: 1, Column: 2},
				R:   '$',
			},
		},
		{
			name: "invalid character after slash",
			src:  `/xfoo`,
			expectedErr: UnexpectedCharacterError{
				Pos: Pos{Line: 1, Column: 3},
				R:   'x',
			},
		},
		{
			name: "unterminated string literal",
			src:  `"hello`,
			expectedErr: UnterminatedStringError{
				Pos: Pos{Line: 1, Column: 1},
			},
		},
		{
			name: "bare minus sign",
			src:  `- x`,
			expectedErr: InvalidNumberError{
				Pos:   Pos{Line: 1, Column: 1},
				Value: "-",
			},
		},
		{
			name: "trailing dot on number",
			src:  `3. `,
			expectedErr: InvalidNumberError{
				Pos:   Pos{Line: 1, Column: 1},
				Value: "3.",
			},
		},
		{
			name: "minus dot without digits",
			src:  `-.5`,
			expectedErr: InvalidNumberError{
				Pos:   Pos{Line: 1, Column: 1},
				Value: "-.5",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			collect := func(seq iter.Seq2[Token, error]) ([]Token, error) {
				var tokens []Token
				for item, err := range seq {
					if err != nil {
						return tokens, err
					}
					tokens = append(tokens, item)
				}
				return tokens, nil
			}

			_, err := collect(Tokenize(strings.NewReader(tc.src)))

			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestTokenizer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		src      string
		expected []Token
	}{
		{
			name: "primitive schema with default namespace",
			src:  `schema int;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "primitive schema with custom namespace",
			src: `namespace com.example;
schema int;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("namespace")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenIdentifier, Value: []byte("com.example")},
				{Pos: Pos{Line: 1, Column: 22}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 2, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "primitive schema with single line comment",
			src: `// This is a comment
schema int;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenComment, Value: []byte("// This is a comment")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 2, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "primitive schema with multi single line comment",
			src: `/* This is a comment */
schema int;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenComment, Value: []byte("/* This is a comment */")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 2, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "primitive schema with multi multi line comment",
			src: `/*
* This is a comment
*/
schema int;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenComment, Value: []byte("/*\n* This is a comment\n*/")},
				{Pos: Pos{Line: 4, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 4, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 4, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "schema with enum type",
			src: `schema int;
enum Suit { HEARTS, DIAMONDS, CLUBS, SPADES };`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("enum")},
				{Pos: Pos{Line: 2, Column: 6}, Type: TokenIdentifier, Value: []byte("Suit")},
				{Pos: Pos{Line: 2, Column: 11}, Type: TokenSymbol, Value: []byte("{")},
				{Pos: Pos{Line: 2, Column: 13}, Type: TokenIdentifier, Value: []byte("HEARTS")},
				{Pos: Pos{Line: 2, Column: 19}, Type: TokenSymbol, Value: []byte(",")},
				{Pos: Pos{Line: 2, Column: 21}, Type: TokenIdentifier, Value: []byte("DIAMONDS")},
				{Pos: Pos{Line: 2, Column: 29}, Type: TokenSymbol, Value: []byte(",")},
				{Pos: Pos{Line: 2, Column: 31}, Type: TokenIdentifier, Value: []byte("CLUBS")},
				{Pos: Pos{Line: 2, Column: 36}, Type: TokenSymbol, Value: []byte(",")},
				{Pos: Pos{Line: 2, Column: 38}, Type: TokenIdentifier, Value: []byte("SPADES")},
				{Pos: Pos{Line: 2, Column: 45}, Type: TokenSymbol, Value: []byte("}")},
				{Pos: Pos{Line: 2, Column: 46}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "schema with fixed type",
			src: `schema int;
fixed MD5(16);`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("fixed")},
				{Pos: Pos{Line: 2, Column: 7}, Type: TokenIdentifier, Value: []byte("MD5")},
				{Pos: Pos{Line: 2, Column: 10}, Type: TokenSymbol, Value: []byte("(")},
				{Pos: Pos{Line: 2, Column: 11}, Type: TokenNumber, Value: []byte("16")},
				{Pos: Pos{Line: 2, Column: 13}, Type: TokenSymbol, Value: []byte(")")},
				{Pos: Pos{Line: 2, Column: 14}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "schema with map type",
			src:  `schema map<int>;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("map")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte("<")},
				{Pos: Pos{Line: 1, Column: 12}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 15}, Type: TokenSymbol, Value: []byte(">")},
				{Pos: Pos{Line: 1, Column: 16}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "schema with union type",
			src:  `schema union { null, string };`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("union")},
				{Pos: Pos{Line: 1, Column: 14}, Type: TokenSymbol, Value: []byte("{")},
				{Pos: Pos{Line: 1, Column: 16}, Type: TokenIdentifier, Value: []byte("null")},
				{Pos: Pos{Line: 1, Column: 20}, Type: TokenSymbol, Value: []byte(",")},
				{Pos: Pos{Line: 1, Column: 22}, Type: TokenIdentifier, Value: []byte("string")},
				{Pos: Pos{Line: 1, Column: 29}, Type: TokenSymbol, Value: []byte("}")},
				{Pos: Pos{Line: 1, Column: 30}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "schema with nullable shorthand",
			src:  `schema string?;`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("string")},
				{Pos: Pos{Line: 1, Column: 14}, Type: TokenSymbol, Value: []byte("?")},
				{Pos: Pos{Line: 1, Column: 15}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "schema with record type with single field",
			src: `schema int;
record Employee {
  string name;
};`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("record")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("Employee")},
				{Pos: Pos{Line: 2, Column: 17}, Type: TokenSymbol, Value: []byte("{")},
				{Pos: Pos{Line: 3, Column: 3}, Type: TokenIdentifier, Value: []byte("string")},
				{Pos: Pos{Line: 3, Column: 10}, Type: TokenIdentifier, Value: []byte("name")},
				{Pos: Pos{Line: 3, Column: 14}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 4, Column: 1}, Type: TokenSymbol, Value: []byte("}")},
				{Pos: Pos{Line: 4, Column: 2}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
		{
			name: "string literal",
			src:  `"hello"`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenString, Value: []byte("hello")},
			},
		},
		{
			name: "empty string literal",
			src:  `""`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenString, Value: nil},
			},
		},
		{
			name: "string literal with escape sequences",
			src:  `"say \"hi\""`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenString, Value: []byte(`say \"hi\"`)},
			},
		},
		{
			name: "decimal number",
			src:  `45.67`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenNumber, Value: []byte("45.67")},
			},
		},
		{
			name: "negative integer",
			src:  `-1`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenNumber, Value: []byte("-1")},
			},
		},
		{
			name: "negative decimal number",
			src:  `-3.14`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenNumber, Value: []byte("-3.14")},
			},
		},
		{
			name: "colon symbol",
			src:  `"key" : "value"`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenString, Value: []byte("key")},
				{Pos: Pos{Line: 1, Column: 7}, Type: TokenSymbol, Value: []byte(":")},
				{Pos: Pos{Line: 1, Column: 9}, Type: TokenString, Value: []byte("value")},
			},
		},
		{
			name: "record field with string default value",
			src: `schema int;
record Employee {
  string name = "unknown";
}`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("record")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("Employee")},
				{Pos: Pos{Line: 2, Column: 17}, Type: TokenSymbol, Value: []byte("{")},
				{Pos: Pos{Line: 3, Column: 3}, Type: TokenIdentifier, Value: []byte("string")},
				{Pos: Pos{Line: 3, Column: 10}, Type: TokenIdentifier, Value: []byte("name")},
				{Pos: Pos{Line: 3, Column: 15}, Type: TokenSymbol, Value: []byte("=")},
				{Pos: Pos{Line: 3, Column: 17}, Type: TokenString, Value: []byte("unknown")},
				{Pos: Pos{Line: 3, Column: 26}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 4, Column: 1}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "record field with integer default value",
			src: `schema int;
record Counter {
  int count = 42;
}`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("record")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("Counter")},
				{Pos: Pos{Line: 2, Column: 16}, Type: TokenSymbol, Value: []byte("{")},
				{Pos: Pos{Line: 3, Column: 3}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 3, Column: 7}, Type: TokenIdentifier, Value: []byte("count")},
				{Pos: Pos{Line: 3, Column: 13}, Type: TokenSymbol, Value: []byte("=")},
				{Pos: Pos{Line: 3, Column: 15}, Type: TokenNumber, Value: []byte("42")},
				{Pos: Pos{Line: 3, Column: 17}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 4, Column: 1}, Type: TokenSymbol, Value: []byte("}")},
			},
		},
		{
			name: "annotation with simple name",
			src:  `@order`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenAnnotation, Value: []byte("order")},
			},
		},
		{
			name: "annotation with dashes in name",
			src:  `@java-class`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenAnnotation, Value: []byte("java-class")},
			},
		},
		{
			name: "annotation with dots in name",
			src:  `@my.custom.prop`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenAnnotation, Value: []byte("my.custom.prop")},
			},
		},
		{
			name: "annotation with string value before identifier",
			src:  `@namespace("org.foo") record`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenAnnotation, Value: []byte("namespace")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte("(")},
				{Pos: Pos{Line: 1, Column: 12}, Type: TokenString, Value: []byte("org.foo")},
				{Pos: Pos{Line: 1, Column: 21}, Type: TokenSymbol, Value: []byte(")")},
				{Pos: Pos{Line: 1, Column: 23}, Type: TokenIdentifier, Value: []byte("record")},
			},
		},
		{
			name: "annotation with array value",
			src:  `@aliases(["old", "ancient"])`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenAnnotation, Value: []byte("aliases")},
				{Pos: Pos{Line: 1, Column: 9}, Type: TokenSymbol, Value: []byte("(")},
				{Pos: Pos{Line: 1, Column: 10}, Type: TokenSymbol, Value: []byte("[")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenString, Value: []byte("old")},
				{Pos: Pos{Line: 1, Column: 16}, Type: TokenSymbol, Value: []byte(",")},
				{Pos: Pos{Line: 1, Column: 18}, Type: TokenString, Value: []byte("ancient")},
				{Pos: Pos{Line: 1, Column: 27}, Type: TokenSymbol, Value: []byte("]")},
				{Pos: Pos{Line: 1, Column: 28}, Type: TokenSymbol, Value: []byte(")")},
			},
		},
		{
			name: "schema with record type with nullable field",
			src: `schema int;
record Employee {
  string name;
  int? age;
};`,
			expected: []Token{
				{Pos: Pos{Line: 1, Column: 1}, Type: TokenIdentifier, Value: []byte("schema")},
				{Pos: Pos{Line: 1, Column: 8}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 1, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 2, Column: 1}, Type: TokenIdentifier, Value: []byte("record")},
				{Pos: Pos{Line: 2, Column: 8}, Type: TokenIdentifier, Value: []byte("Employee")},
				{Pos: Pos{Line: 2, Column: 17}, Type: TokenSymbol, Value: []byte("{")},
				{Pos: Pos{Line: 3, Column: 3}, Type: TokenIdentifier, Value: []byte("string")},
				{Pos: Pos{Line: 3, Column: 10}, Type: TokenIdentifier, Value: []byte("name")},
				{Pos: Pos{Line: 3, Column: 14}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 4, Column: 3}, Type: TokenIdentifier, Value: []byte("int")},
				{Pos: Pos{Line: 4, Column: 6}, Type: TokenSymbol, Value: []byte("?")},
				{Pos: Pos{Line: 4, Column: 8}, Type: TokenIdentifier, Value: []byte("age")},
				{Pos: Pos{Line: 4, Column: 11}, Type: TokenSymbol, Value: []byte(";")},
				{Pos: Pos{Line: 5, Column: 1}, Type: TokenSymbol, Value: []byte("}")},
				{Pos: Pos{Line: 5, Column: 2}, Type: TokenSymbol, Value: []byte(";")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			collect := func(seq iter.Seq2[Token, error]) ([]Token, error) {
				tokens := make([]Token, 0, len(tc.expected))
				for item, err := range seq {
					if err != nil {
						return tokens, err
					}
					t.Log(item)
					tokens = append(tokens, item)
				}
				return tokens, nil
			}

			tokens, err := collect(Tokenize(strings.NewReader(tc.src)))

			require.NoError(t, err)
			require.Equal(t, tc.expected, tokens)
		})
	}
}
