// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type binaryMarshalerFunc func(w *BinaryWriter) error

func (f binaryMarshalerFunc) MarshalAvroBinary(w *BinaryWriter) error {
	return f(w)
}

func TestMarshalBinary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		v        BinaryMarshaler
		expected []byte
	}{
		{
			name: "write bool true",
			v: binaryMarshalerFunc(func(w *BinaryWriter) error {
				return w.WriteBool(true)
			}),
			expected: []byte{0x01},
		},
		{
			name: "write string",
			v: binaryMarshalerFunc(func(w *BinaryWriter) error {
				return w.WriteString("abc")
			}),
			expected: []byte{0x06, 0x61, 0x62, 0x63},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			err := MarshalBinary(&buf, tc.v)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestWriteBool(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    bool
		expected []byte
	}{
		{
			name:     "true",
			input:    true,
			expected: []byte{0x01},
		},
		{
			name:     "false",
			input:    false,
			expected: []byte{0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteBool(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

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

func TestWriteFloat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    float32
		expected []byte
	}{
		{
			name:  "zero",
			input: 0.0,
			expected: func() []byte {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, math.Float32bits(0.0))
				return buf
			}(),
		},
		{
			name:  "positive",
			input: 3.14,
			expected: func() []byte {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, math.Float32bits(3.14))
				return buf
			}(),
		},
		{
			name:  "negative",
			input: -1.5,
			expected: func() []byte {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, math.Float32bits(-1.5))
				return buf
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteFloat(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestWriteDouble(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    float64
		expected []byte
	}{
		{
			name:  "zero",
			input: 0.0,
			expected: func() []byte {
				buf := make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, math.Float64bits(0.0))
				return buf
			}(),
		},
		{
			name:  "positive",
			input: 3.14,
			expected: func() []byte {
				buf := make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, math.Float64bits(3.14))
				return buf
			}(),
		},
		{
			name:  "negative",
			input: -1.5,
			expected: func() []byte {
				buf := make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, math.Float64bits(-1.5))
				return buf
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteDouble(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestWriteBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "empty",
			input:    []byte{},
			expected: []byte{0x00},
		},
		{
			name:     "single byte",
			input:    []byte{0xff},
			expected: []byte{0x02, 0xff},
		},
		{
			name:     "multiple bytes",
			input:    []byte{0x01, 0x02, 0x03},
			expected: []byte{0x06, 0x01, 0x02, 0x03},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteBytes(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestWriteString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "empty",
			input:    "",
			expected: []byte{0x00},
		},
		{
			name:     "ascii",
			input:    "abc",
			expected: []byte{0x06, 0x61, 0x62, 0x63},
		},
		{
			name:     "utf8 multibyte",
			input:    "\u00e9",
			expected: []byte{0x04, 0xc3, 0xa9},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteString(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}
