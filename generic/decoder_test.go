// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/z5labs/avro-go"
)

// decodeOne is a small helper that validates s, then decodes data into a
// Value.
func decodeOne(t *testing.T, s avro.Schema, data []byte) (Value, error) {
	t.Helper()
	dec, err := NewDecoder(s)
	if err != nil {
		return nil, err
	}
	return dec.Decode(bytes.NewReader(data))
}

func TestDecoder_Primitives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		schema avro.Schema
		data   []byte
		want   Value
	}{
		{name: "null", schema: avro.Null{}, data: nil, want: Null{}},
		{name: "boolean true", schema: avro.Boolean{}, data: []byte{0x01}, want: Bool(true)},
		{name: "boolean false", schema: avro.Boolean{}, data: []byte{0x00}, want: Bool(false)},
		{name: "int 1", schema: avro.Int{}, data: []byte{0x02}, want: Int(1)},
		{name: "int -1", schema: avro.Int{}, data: []byte{0x01}, want: Int(-1)},
		{name: "long 64", schema: avro.Long{}, data: []byte{0x80, 0x01}, want: Long(64)},
		{name: "string abc", schema: avro.String{}, data: []byte{0x06, 'a', 'b', 'c'}, want: String("abc")},
		{name: "bytes", schema: avro.Bytes{}, data: []byte{0x04, 0xde, 0xad}, want: Bytes{0xde, 0xad}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeOne(t, tc.schema, tc.data)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDecoder_RuntimeErrors(t *testing.T) {
	t.Parallel()

	t.Run("union index out of range", func(t *testing.T) {
		t.Parallel()
		_, err := decodeOne(t, avro.Union{Types: []avro.Schema{avro.Null{}, avro.Int{}}}, []byte{0x06})
		var got IndexOutOfRangeError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "union", got.Kind)
		require.Equal(t, int64(3), got.Index)
		require.Equal(t, 2, got.Len)
	})

	t.Run("enum index out of range", func(t *testing.T) {
		t.Parallel()
		_, err := decodeOne(t, avro.Enum{Name: "E", Symbols: []string{"A", "B"}}, []byte{0x06})
		var got IndexOutOfRangeError
		require.ErrorAs(t, err, &got)
		require.Equal(t, "enum", got.Kind)
		require.Equal(t, "E", got.Name)
		require.Equal(t, int64(3), got.Index)
		require.Equal(t, 2, got.Len)
	})
}

func TestDecoder_IndexOutOfRange(t *testing.T) {
	t.Parallel()

	enum := avro.Enum{Name: "E", Symbols: []string{"A", "B"}}
	union := avro.Union{Types: []avro.Schema{avro.Null{}, avro.Int{}}}

	testCases := []struct {
		name        string
		schema      avro.Schema
		data        []byte
		expected    IndexOutOfRangeError
		expectedMsg string
	}{
		{
			name:        "enum ordinal equal to symbol count",
			schema:      enum,
			data:        []byte{0x04},
			expected:    IndexOutOfRangeError{Kind: "enum", Name: "E", Index: 2, Len: 2},
			expectedMsg: `avro/generic: enum "E": index 2 out of range [0, 2)`,
		},
		{
			name:        "enum ordinal above symbol count",
			schema:      enum,
			data:        []byte{0x06},
			expected:    IndexOutOfRangeError{Kind: "enum", Name: "E", Index: 3, Len: 2},
			expectedMsg: `avro/generic: enum "E": index 3 out of range [0, 2)`,
		},
		{
			name:        "negative enum ordinal",
			schema:      enum,
			data:        []byte{0x01},
			expected:    IndexOutOfRangeError{Kind: "enum", Name: "E", Index: -1, Len: 2},
			expectedMsg: `avro/generic: enum "E": index -1 out of range [0, 2)`,
		},
		{
			name:        "union branch equal to branch count",
			schema:      union,
			data:        []byte{0x04},
			expected:    IndexOutOfRangeError{Kind: "union", Index: 2, Len: 2},
			expectedMsg: "avro/generic: union: index 2 out of range [0, 2)",
		},
		{
			name:        "union branch above branch count",
			schema:      union,
			data:        []byte{0x06},
			expected:    IndexOutOfRangeError{Kind: "union", Index: 3, Len: 2},
			expectedMsg: "avro/generic: union: index 3 out of range [0, 2)",
		},
		{
			name:        "negative union branch",
			schema:      union,
			data:        []byte{0x01},
			expected:    IndexOutOfRangeError{Kind: "union", Index: -1, Len: 2},
			expectedMsg: "avro/generic: union: index -1 out of range [0, 2)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := decodeOne(t, tc.schema, tc.data)

			require.ErrorIs(t, err, avro.ErrInvalidIndex)
			var got IndexOutOfRangeError
			require.ErrorAs(t, err, &got)
			require.Equal(t, tc.expected, got)
			require.EqualError(t, err, tc.expectedMsg)
		})
	}
}

func TestIndexOutOfRangeError_Is(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		target   error
		expected bool
	}{
		{
			name:     "matches avro.ErrInvalidIndex",
			target:   avro.ErrInvalidIndex,
			expected: true,
		},
		{
			name:     "does not match avro.ErrOverflow",
			target:   avro.ErrOverflow,
			expected: false,
		},
		{
			name:     "does not match an unrelated error",
			target:   errors.New("avro: enum or union index out of range"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := error(IndexOutOfRangeError{Kind: "enum", Name: "E", Index: 3, Len: 2})

			require.Equal(t, tc.expected, errors.Is(err, tc.target))
		})
	}
}
