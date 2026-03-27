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
			name:        "namespace missing semicolon",
			src:         "namespace com.example ",
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenSymbol}},
		},
		{
			name:        "namespace with no following tokens",
			src:         `namespace com.example;`,
			expectedErr: UnexpectedEndOfTokensError{Expected: []TokenType{TokenIdentifier, TokenComment}},
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
