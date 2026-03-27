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
