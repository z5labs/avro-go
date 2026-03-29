// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFingerprint64(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: 0xc15d213aa4d7a795,
		},
		{
			name:     "null schema",
			input:    []byte(`"null"`),
			expected: 0x63dd24e7cc258f8a,
		},
		{
			name:     "boolean schema",
			input:    []byte(`"boolean"`),
			expected: 0x9f42fc78a4d4f764,
		},
		{
			name:     "int schema",
			input:    []byte(`"int"`),
			expected: 0x7275d51a3f395c8f,
		},
		{
			name:     "long schema",
			input:    []byte(`"long"`),
			expected: 0xd054e14493f41db7,
		},
		{
			name:     "float schema",
			input:    []byte(`"float"`),
			expected: 0x4d7c02cb3ea8d790,
		},
		{
			name:     "double schema",
			input:    []byte(`"double"`),
			expected: 0x8e7535c032ab957e,
		},
		{
			name:     "bytes schema",
			input:    []byte(`"bytes"`),
			expected: 0x4fc016dac3201965,
		},
		{
			name:     "string schema",
			input:    []byte(`"string"`),
			expected: 0x8f014872634503c7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := Fingerprint64(tc.input)

			require.Equal(t, tc.expected, result)
		})
	}
}
