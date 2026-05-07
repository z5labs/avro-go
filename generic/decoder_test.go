// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"bytes"
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
