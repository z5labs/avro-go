// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    int32
		expected []byte
	}{
		{
			name:     "zero",
			input:    0,
			expected: []byte{0x00},
		},
		{
			name:     "positive one",
			input:    1,
			expected: []byte{0x02},
		},
		{
			name:     "negative one",
			input:    -1,
			expected: []byte{0x01},
		},
		{
			name:     "positive two",
			input:    2,
			expected: []byte{0x04},
		},
		{
			name:     "negative two",
			input:    -2,
			expected: []byte{0x03},
		},
		{
			name:     "sixty four",
			input:    64,
			expected: []byte{0x80, 0x01},
		},
		{
			name:     "max int32",
			input:    math.MaxInt32,
			expected: []byte{0xfe, 0xff, 0xff, 0xff, 0x0f},
		},
		{
			name:     "min int32",
			input:    math.MinInt32,
			expected: []byte{0xff, 0xff, 0xff, 0xff, 0x0f},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteInt(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestWriteLong(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    int64
		expected []byte
	}{
		{
			name:     "zero",
			input:    0,
			expected: []byte{0x00},
		},
		{
			name:     "positive one",
			input:    1,
			expected: []byte{0x02},
		},
		{
			name:     "negative one",
			input:    -1,
			expected: []byte{0x01},
		},
		{
			name:     "positive two",
			input:    2,
			expected: []byte{0x04},
		},
		{
			name:     "negative two",
			input:    -2,
			expected: []byte{0x03},
		},
		{
			name:     "sixty four",
			input:    64,
			expected: []byte{0x80, 0x01},
		},
		{
			name:     "max int64",
			input:    math.MaxInt64,
			expected: []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
		},
		{
			name:     "min int64",
			input:    math.MinInt64,
			expected: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteLong(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}
