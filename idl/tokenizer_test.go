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
