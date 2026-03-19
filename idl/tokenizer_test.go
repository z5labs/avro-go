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

func TestTokenizer(t *testing.T) {
	testCases := []struct {
		name     string
		src      string
		expected []Token
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
			src: `/*
* This is a comment
*/
schema int;`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := make([]Token, 0, len(tc.expected))
			for token, err := range Tokenize(strings.NewReader(tc.src)) {
				require.Nil(t, err)
				tokens = append(tokens, token)
			}

			require.Equal(t, tc.expected, tokens)
		})
	}
}
