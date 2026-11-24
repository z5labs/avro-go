// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) {
	return f(p)
}

type binaryMarshalerFunc func(w *BinaryWriter) error

func (f binaryMarshalerFunc) MarshalAvroBinary(w *BinaryWriter) error {
	return f(w)
}

type binaryUnmarshalerFunc func(r *BinaryReader) error

func (f binaryUnmarshalerFunc) UnmarshalAvroBinary(r *BinaryReader) error {
	return f(r)
}

func TestUnmarshalBinary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []byte
		v     BinaryUnmarshaler
	}{
		{
			name:  "read bool true",
			input: []byte{0x01},
			v: binaryUnmarshalerFunc(func(r *BinaryReader) error {
				_, err := r.ReadBool()
				return err
			}),
		},
		{
			name:  "read string",
			input: []byte{0x06, 0x61, 0x62, 0x63},
			v: binaryUnmarshalerFunc(func(r *BinaryReader) error {
				_, err := r.ReadString()
				return err
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := UnmarshalBinary(bytes.NewReader(tc.input), tc.v)

			require.NoError(t, err)
		})
	}
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

func TestWriteBool_error(t *testing.T) {
	t.Parallel()

	t.Run("write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteBool(true)

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("short write", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, nil
		})}

		err := w.WriteBool(true)

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
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

func TestWriteInt_error(t *testing.T) {
	t.Parallel()

	t.Run("write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteInt(1)

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
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

func TestWriteLong_error(t *testing.T) {
	t.Parallel()

	t.Run("write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteLong(1)

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("short write", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, nil
		})}

		err := w.WriteLong(1)

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
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

func TestWriteFloat_error(t *testing.T) {
	t.Parallel()

	t.Run("write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteFloat(3.14)

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("short write", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, nil
		})}

		err := w.WriteFloat(3.14)

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
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

func TestWriteDouble_error(t *testing.T) {
	t.Parallel()

	t.Run("write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteDouble(3.14)

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("short write", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, nil
		})}

		err := w.WriteDouble(3.14)

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
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

func TestWriteBytes_error(t *testing.T) {
	t.Parallel()

	t.Run("length write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteBytes([]byte{0x01})

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("data write error", func(t *testing.T) {
		t.Parallel()

		calls := 0
		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				return len(p), nil
			}
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteBytes([]byte{0x01})

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("data short write", func(t *testing.T) {
		t.Parallel()

		calls := 0
		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				return len(p), nil
			}
			return 0, nil
		})}

		err := w.WriteBytes([]byte{0x01})

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
}

func TestWriteFixed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "single byte",
			input:    []byte{0xff},
			expected: []byte{0xff},
		},
		{
			name:     "multiple bytes",
			input:    []byte{0x01, 0x02, 0x03},
			expected: []byte{0x01, 0x02, 0x03},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			w := &BinaryWriter{out: &buf}

			err := w.WriteFixed(tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestWriteFixed_error(t *testing.T) {
	t.Parallel()

	t.Run("write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteFixed([]byte{0x01, 0x02})

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("short write", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, nil
		})}

		err := w.WriteFixed([]byte{0x01, 0x02})

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
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

func TestWriteString_error(t *testing.T) {
	t.Parallel()

	t.Run("length write error", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteString("abc")

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("data write error", func(t *testing.T) {
		t.Parallel()

		calls := 0
		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				return len(p), nil
			}
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteString("abc")

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("data short write", func(t *testing.T) {
		t.Parallel()

		calls := 0
		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				return len(p), nil
			}
			return 0, nil
		})}

		err := w.WriteString("abc")

		require.ErrorIs(t, err, io.ErrShortWrite)
	})
}

func TestReadBool(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "true",
			input:    []byte{0x01},
			expected: true,
		},
		{
			name:     "false",
			input:    []byte{0x00},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadBool()

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadBool_error(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadBool()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       []byte
		expected    int32
		expectedErr error
	}{
		{
			name:     "zero",
			input:    []byte{0x00},
			expected: 0,
		},
		{
			name:     "positive one",
			input:    []byte{0x02},
			expected: 1,
		},
		{
			name:     "negative one",
			input:    []byte{0x01},
			expected: -1,
		},
		{
			name:     "positive two",
			input:    []byte{0x04},
			expected: 2,
		},
		{
			name:     "negative two",
			input:    []byte{0x03},
			expected: -2,
		},
		{
			name:     "sixty four",
			input:    []byte{0x80, 0x01},
			expected: 64,
		},
		{
			name:     "max int32",
			input:    []byte{0xfe, 0xff, 0xff, 0xff, 0x0f},
			expected: math.MaxInt32,
		},
		{
			name:     "min int32",
			input:    []byte{0xff, 0xff, 0xff, 0xff, 0x0f},
			expected: math.MinInt32,
		},
		{
			name:        "overflow",
			input:       []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
			expectedErr: ErrOverflow,
		},
		{
			name:        "tenth byte of 0x02 overflows",
			input:       []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02},
			expectedErr: ErrOverflow,
		},
		{
			name:        "tenth byte of 0x7f overflows",
			input:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
			expectedErr: ErrOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadInt()

			if tc.expectedErr != nil {
				require.True(t, errors.Is(err, tc.expectedErr))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadInt_error(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadInt()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadLong(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       []byte
		expected    int64
		expectedErr error
	}{
		{
			name:     "zero",
			input:    []byte{0x00},
			expected: 0,
		},
		{
			name:     "positive one",
			input:    []byte{0x02},
			expected: 1,
		},
		{
			name:     "negative one",
			input:    []byte{0x01},
			expected: -1,
		},
		{
			name:     "positive two",
			input:    []byte{0x04},
			expected: 2,
		},
		{
			name:     "negative two",
			input:    []byte{0x03},
			expected: -2,
		},
		{
			name:     "sixty four",
			input:    []byte{0x80, 0x01},
			expected: 64,
		},
		{
			name:     "max int64",
			input:    []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
			expected: math.MaxInt64,
		},
		{
			name:     "min int64",
			input:    []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
			expected: math.MinInt64,
		},
		{
			name:        "overflow",
			input:       []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01},
			expectedErr: ErrOverflow,
		},
		{
			name:        "tenth byte of 0x81 overflows",
			input:       []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x81},
			expectedErr: ErrOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadLong()

			if tc.expectedErr != nil {
				require.True(t, errors.Is(err, tc.expectedErr))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadLong_error(t *testing.T) {
	t.Parallel()

	t.Run("read error on first byte", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadLong()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("read error on continuation byte", func(t *testing.T) {
		t.Parallel()

		calls := 0
		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				p[0] = 0x80
				return 1, nil
			}
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadLong()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadLong_tenthByteOverflow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		tenth       byte
		expected    int64
		expectedErr error
	}{
		{
			name:     "0x01 decodes",
			tenth:    0x01,
			expected: math.MinInt64,
		},
		{
			name:        "0x02 overflows",
			tenth:       0x02,
			expectedErr: ErrOverflow,
		},
		{
			name:        "0x7f overflows",
			tenth:       0x7f,
			expectedErr: ErrOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := append(bytes.Repeat([]byte{0xff}, binary.MaxVarintLen64-1), tc.tenth)
			r := NewBinaryReader(bytes.NewReader(input))

			got, err := r.ReadLong()

			if tc.expectedErr != nil {
				var rerr *BinaryReaderError
				require.ErrorAs(t, err, &rerr)
				require.ErrorIs(t, err, tc.expectedErr)
				require.Equal(t, int64(binary.MaxVarintLen64), rerr.Offset)
				require.Equal(t, int64(binary.MaxVarintLen64), r.Offset())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadIndex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		input          []byte
		count          int
		expected       int
		expectedErr    error
		expectedOffset int64
	}{
		{
			name:           "first index",
			input:          []byte{0x00},
			count:          3,
			expected:       0,
			expectedOffset: 1,
		},
		{
			name:           "last index",
			input:          []byte{0x04},
			count:          3,
			expected:       2,
			expectedOffset: 1,
		},
		{
			name:           "multi byte index",
			input:          []byte{0x90, 0x03},
			count:          201,
			expected:       200,
			expectedOffset: 2,
		},
		{
			name:           "negative index",
			input:          []byte{0x01},
			count:          3,
			expectedErr:    ErrInvalidIndex,
			expectedOffset: 1,
		},
		{
			name:           "min int64 index",
			input:          []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
			count:          3,
			expectedErr:    ErrInvalidIndex,
			expectedOffset: 10,
		},
		{
			name:           "index equal to count",
			input:          []byte{0x06},
			count:          3,
			expectedErr:    ErrInvalidIndex,
			expectedOffset: 1,
		},
		{
			name:           "index above count",
			input:          []byte{0x80, 0x01},
			count:          3,
			expectedErr:    ErrInvalidIndex,
			expectedOffset: 2,
		},
		{
			name:           "max int64 index",
			input:          []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},
			count:          3,
			expectedErr:    ErrInvalidIndex,
			expectedOffset: 10,
		},
		{
			name:           "index of a zero count",
			input:          []byte{0x00},
			count:          0,
			expectedErr:    ErrInvalidIndex,
			expectedOffset: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewBinaryReader(bytes.NewReader(tc.input))

			got, err := r.ReadIndex(tc.count)

			require.Equal(t, tc.expectedOffset, r.Offset())
			if tc.expectedErr != nil {
				var rerr *BinaryReaderError
				require.ErrorAs(t, err, &rerr)
				require.ErrorIs(t, err, tc.expectedErr)
				require.Equal(t, tc.expectedOffset, rerr.Offset)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

// TestReadIndex_varintErrors checks that ReadIndex reports every varint
// failure exactly as ReadLong does, at the same offset.
func TestReadIndex_varintErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty input",
			input: []byte{},
		},
		{
			name:  "continuation byte with nothing after it",
			input: []byte{0x80},
		},
		{
			name:  "cut after nine bytes",
			input: bytes.Repeat([]byte{0xff}, binary.MaxVarintLen64-1),
		},
		{
			name:  "tenth byte of 0x02 overflows",
			input: append(bytes.Repeat([]byte{0xff}, binary.MaxVarintLen64-1), 0x02),
		},
		{
			name:  "eleven byte varint overflows",
			input: []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lr := NewBinaryReader(bytes.NewReader(tc.input))
			_, longErr := lr.ReadLong()
			require.Error(t, longErr)

			ir := NewBinaryReader(bytes.NewReader(tc.input))
			_, indexErr := ir.ReadIndex(3)

			require.Equal(t, longErr, indexErr)
			require.Equal(t, lr.Offset(), ir.Offset())
			require.NotErrorIs(t, indexErr, ErrInvalidIndex)
		})
	}

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := NewBinaryReader(readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		}))

		_, err := r.ReadIndex(3)

		var rerr *BinaryReaderError
		require.ErrorAs(t, err, &rerr)
		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

// truncations returns every proper prefix of data, including the empty one.
// Each is a value cut short at that position.
func truncations(data []byte) [][]byte {
	prefixes := make([][]byte, len(data))
	for i := range prefixes {
		prefixes[i] = data[:i]
	}
	return prefixes
}

// truncationErr is the error a read of a value cut short after n bytes must
// report: io.EOF when no byte of it was available, io.ErrUnexpectedEOF after.
func truncationErr(n int) error {
	if n == 0 {
		return io.EOF
	}
	return io.ErrUnexpectedEOF
}

func TestBinaryReader_truncated(t *testing.T) {
	t.Parallel()

	maxLong := []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
	minInt := []byte{0xff, 0xff, 0xff, 0xff, 0x0f}

	// A 64-byte payload has a two-byte length prefix, so truncation lands
	// inside the prefix as well as after it.
	payload := bytes.Repeat([]byte{'a'}, 64)
	var longBytes bytes.Buffer
	require.NoError(t, NewBinaryWriter(&longBytes).WriteBytes(payload))
	require.Equal(t, 2+len(payload), longBytes.Len())

	readLong := func(r *BinaryReader) error {
		_, err := r.ReadLong()
		return err
	}
	readBytes := func(r *BinaryReader) error {
		_, err := r.ReadBytes()
		return err
	}

	reads := []struct {
		name string
		data []byte
		read func(r *BinaryReader) error
	}{
		{
			name: "ReadLong",
			data: maxLong,
			read: readLong,
		},
		{
			name: "ReadInt",
			data: minInt,
			read: func(r *BinaryReader) error {
				_, err := r.ReadInt()
				return err
			},
		},
		{
			name: "ReadIndex",
			data: maxLong,
			read: func(r *BinaryReader) error {
				_, err := r.ReadIndex(math.MaxInt)
				return err
			},
		},
		{
			name: "ReadBytes",
			data: longBytes.Bytes(),
			read: readBytes,
		},
		{
			name: "ReadString",
			data: longBytes.Bytes(),
			read: func(r *BinaryReader) error {
				_, err := r.ReadString()
				return err
			},
		},
	}

	type testCase struct {
		name           string
		input          []byte
		read           func(r *BinaryReader) error
		expectedErr    error
		expectedOffset int64
	}
	var testCases []testCase
	for _, rd := range reads {
		for _, prefix := range truncations(rd.data) {
			testCases = append(testCases, testCase{
				name:           fmt.Sprintf("%s cut after %d of %d bytes", rd.name, len(prefix), len(rd.data)),
				input:          prefix,
				read:           rd.read,
				expectedErr:    truncationErr(len(prefix)),
				expectedOffset: int64(len(prefix)),
			})
		}
	}
	testCases = append(testCases,
		testCase{
			name:           "ReadLong continuation byte with nothing after it",
			input:          []byte{0x80},
			read:           readLong,
			expectedErr:    io.ErrUnexpectedEOF,
			expectedOffset: 1,
		},
		testCase{
			name:           "ReadBytes length prefix with no payload",
			input:          []byte{0x04},
			read:           readBytes,
			expectedErr:    io.ErrUnexpectedEOF,
			expectedOffset: 1,
		},
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewBinaryReader(bytes.NewReader(tc.input))

			err := tc.read(r)

			var rerr *BinaryReaderError
			require.ErrorAs(t, err, &rerr)
			require.ErrorIs(t, rerr.Err, tc.expectedErr)
			if tc.expectedErr == io.ErrUnexpectedEOF {
				require.NotErrorIs(t, err, io.EOF)
			}
			require.Equal(t, tc.expectedOffset, rerr.Offset)
			require.Equal(t, tc.expectedOffset, r.Offset())
		})
	}
}

func TestReadFloat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected float32
	}{
		{
			name: "zero",
			input: func() []byte {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, math.Float32bits(0.0))
				return buf
			}(),
			expected: 0.0,
		},
		{
			name: "positive",
			input: func() []byte {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, math.Float32bits(3.14))
				return buf
			}(),
			expected: 3.14,
		},
		{
			name: "negative",
			input: func() []byte {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, math.Float32bits(-1.5))
				return buf
			}(),
			expected: -1.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadFloat()

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadFloat_error(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadFloat()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadDouble(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected float64
	}{
		{
			name: "zero",
			input: func() []byte {
				buf := make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, math.Float64bits(0.0))
				return buf
			}(),
			expected: 0.0,
		},
		{
			name: "positive",
			input: func() []byte {
				buf := make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, math.Float64bits(3.14))
				return buf
			}(),
			expected: 3.14,
		},
		{
			name: "negative",
			input: func() []byte {
				buf := make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, math.Float64bits(-1.5))
				return buf
			}(),
			expected: -1.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadDouble()

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadDouble_error(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadDouble()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       []byte
		expected    []byte
		expectedErr error
	}{
		{
			name:     "empty",
			input:    []byte{0x00},
			expected: []byte{},
		},
		{
			name:     "single byte",
			input:    []byte{0x02, 0xff},
			expected: []byte{0xff},
		},
		{
			name:     "multiple bytes",
			input:    []byte{0x06, 0x01, 0x02, 0x03},
			expected: []byte{0x01, 0x02, 0x03},
		},
		{
			name:        "negative length",
			input:       []byte{0x01},
			expectedErr: ErrNegativeLength,
		},
		{
			name:        "overflow length",
			input:       []byte{0x80, 0x80, 0x80, 0x80, 0x10},
			expectedErr: ErrOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadBytes()

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadBytes_error(t *testing.T) {
	t.Parallel()

	t.Run("length read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadBytes()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("data read error", func(t *testing.T) {
		t.Parallel()

		calls := 0
		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				p[0] = 0x02
				return 1, nil
			}
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadBytes()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadFixed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		size     int
		expected []byte
	}{
		{
			name:     "single byte",
			input:    []byte{0xff},
			size:     1,
			expected: []byte{0xff},
		},
		{
			name:     "multiple bytes",
			input:    []byte{0x01, 0x02, 0x03},
			size:     3,
			expected: []byte{0x01, 0x02, 0x03},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadFixed(tc.size)

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadFixed_error(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadFixed(4)

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestReadString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty",
			input:    []byte{0x00},
			expected: "",
		},
		{
			name:     "ascii",
			input:    []byte{0x06, 0x61, 0x62, 0x63},
			expected: "abc",
		},
		{
			name:     "utf8 multibyte",
			input:    []byte{0x04, 0xc3, 0xa9},
			expected: "\u00e9",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &BinaryReader{in: bytes.NewReader(tc.input)}

			got, err := r.ReadString()

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestReadString_error(t *testing.T) {
	t.Parallel()

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		_, err := r.ReadString()

		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestBinaryReader_Offset(t *testing.T) {
	t.Parallel()

	t.Run("zero on fresh reader", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader(nil)}

		require.Equal(t, int64(0), r.Offset())
	})

	t.Run("advances on ReadBool", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x01})}

		_, err := r.ReadBool()
		require.NoError(t, err)
		require.Equal(t, int64(1), r.Offset())
	})

	t.Run("advances on ReadFloat", func(t *testing.T) {
		t.Parallel()

		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, math.Float32bits(1.5))
		r := &BinaryReader{in: bytes.NewReader(buf)}

		_, err := r.ReadFloat()
		require.NoError(t, err)
		require.Equal(t, int64(4), r.Offset())
	})

	t.Run("advances on ReadDouble", func(t *testing.T) {
		t.Parallel()

		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(1.5))
		r := &BinaryReader{in: bytes.NewReader(buf)}

		_, err := r.ReadDouble()
		require.NoError(t, err)
		require.Equal(t, int64(8), r.Offset())
	})

	t.Run("advances on ReadFixed", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04, 0x05})}

		_, err := r.ReadFixed(5)
		require.NoError(t, err)
		require.Equal(t, int64(5), r.Offset())
	})

	t.Run("advances on single-byte ReadLong", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x02})}

		v, err := r.ReadLong()
		require.NoError(t, err)
		require.Equal(t, int64(1), v)
		require.Equal(t, int64(1), r.Offset())
	})

	t.Run("advances on multi-byte ReadLong", func(t *testing.T) {
		t.Parallel()

		var buf [binary.MaxVarintLen64]byte
		n := binary.PutVarint(buf[:], 1<<28)
		require.Equal(t, 5, n)
		r := &BinaryReader{in: bytes.NewReader(buf[:n])}

		_, err := r.ReadLong()
		require.NoError(t, err)
		require.Equal(t, int64(5), r.Offset())
	})

	t.Run("advances on ReadBytes", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x06, 0x61, 0x62, 0x63})}

		got, err := r.ReadBytes()
		require.NoError(t, err)
		require.Equal(t, []byte{0x61, 0x62, 0x63}, got)
		require.Equal(t, int64(4), r.Offset())
	})

	t.Run("advances on ReadString", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x06, 0x61, 0x62, 0x63})}

		got, err := r.ReadString()
		require.NoError(t, err)
		require.Equal(t, "abc", got)
		require.Equal(t, int64(4), r.Offset())
	})

	t.Run("accumulates across multiple reads", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x01, 0x02, 0xff, 0xff, 0xff})}

		_, err := r.ReadBool()
		require.NoError(t, err)
		require.Equal(t, int64(1), r.Offset())

		_, err = r.ReadLong()
		require.NoError(t, err)
		require.Equal(t, int64(2), r.Offset())

		_, err = r.ReadFixed(3)
		require.NoError(t, err)
		require.Equal(t, int64(5), r.Offset())
	})
}

func TestBinaryReader_Offset_WrapsErrors(t *testing.T) {
	t.Parallel()

	t.Run("EOF on empty reader annotates offset 0", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader(nil)}

		_, err := r.ReadBool()

		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		require.Contains(t, err.Error(), "offset 0")
		require.Equal(t, int64(0), r.Offset())
	})

	t.Run("partial read advances offset before wrapping", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			p[0] = 0x01
			p[1] = 0x02
			return 2, io.EOF
		})}

		_, err := r.ReadFloat()

		require.Error(t, err)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		require.Contains(t, err.Error(), "offset 2")
		require.Equal(t, int64(2), r.Offset())
	})

	t.Run("ReadLong varint overflow wraps ErrOverflow with offset", func(t *testing.T) {
		t.Parallel()

		input := make([]byte, binary.MaxVarintLen64)
		for i := range input {
			input[i] = 0x80
		}
		r := &BinaryReader{in: bytes.NewReader(input)}

		_, err := r.ReadLong()

		require.Error(t, err)
		require.ErrorIs(t, err, ErrOverflow)
		require.Contains(t, err.Error(), "offset 10")
		require.Equal(t, int64(10), r.Offset())
	})

	t.Run("ReadBytes negative length wraps with offset after length varint", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: bytes.NewReader([]byte{0x01})}

		_, err := r.ReadBytes()

		require.Error(t, err)
		require.ErrorIs(t, err, ErrNegativeLength)
		require.Contains(t, err.Error(), "offset 1")
		require.Equal(t, int64(1), r.Offset())
	})

	t.Run("ReadInt overflow from out-of-range long wraps with offset", func(t *testing.T) {
		t.Parallel()

		var buf [binary.MaxVarintLen64]byte
		n := binary.PutVarint(buf[:], int64(math.MaxInt32)+1)
		r := &BinaryReader{in: bytes.NewReader(buf[:n])}

		_, err := r.ReadInt()

		require.Error(t, err)
		require.ErrorIs(t, err, ErrOverflow)
		require.Contains(t, err.Error(), "offset")
		require.Equal(t, int64(n), r.Offset())
	})

	t.Run("errors.As extracts BinaryReaderError with offset", func(t *testing.T) {
		t.Parallel()

		r := &BinaryReader{in: readerFunc(func(p []byte) (int, error) {
			p[0] = 0x01
			p[1] = 0x02
			return 2, io.EOF
		})}

		_, err := r.ReadFloat()

		var rerr *BinaryReaderError
		require.ErrorAs(t, err, &rerr)
		require.Equal(t, int64(2), rerr.Offset)
		require.ErrorIs(t, rerr.Err, io.ErrUnexpectedEOF)
	})
}

func TestBinaryWriter_Offset(t *testing.T) {
	t.Parallel()

	t.Run("zero on fresh writer", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.Equal(t, int64(0), w.Offset())
	})

	t.Run("advances on WriteBool", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteBool(true))
		require.Equal(t, int64(1), w.Offset())
	})

	t.Run("advances on WriteFloat", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteFloat(1.5))
		require.Equal(t, int64(4), w.Offset())
	})

	t.Run("advances on WriteDouble", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteDouble(1.5))
		require.Equal(t, int64(8), w.Offset())
	})

	t.Run("advances on WriteFixed", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteFixed([]byte{0x01, 0x02, 0x03}))
		require.Equal(t, int64(3), w.Offset())
	})

	t.Run("advances on multi-byte WriteLong", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteLong(1<<28))
		require.Equal(t, int64(5), w.Offset())
	})

	t.Run("advances on WriteBytes", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteBytes([]byte("abc")))
		require.Equal(t, int64(4), w.Offset())
	})

	t.Run("advances on WriteString", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteString("abc"))
		require.Equal(t, int64(4), w.Offset())
	})

	t.Run("accumulates across multiple writes", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: &bytes.Buffer{}}

		require.NoError(t, w.WriteBool(true))
		require.Equal(t, int64(1), w.Offset())

		require.NoError(t, w.WriteLong(1))
		require.Equal(t, int64(2), w.Offset())

		require.NoError(t, w.WriteFixed([]byte{0x01, 0x02, 0x03}))
		require.Equal(t, int64(5), w.Offset())
	})
}

func TestBinaryWriter_Offset_WrapsErrors(t *testing.T) {
	t.Parallel()

	t.Run("write error annotates offset 0", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		})}

		err := w.WriteBool(true)

		require.Error(t, err)
		require.ErrorIs(t, err, io.ErrClosedPipe)
		require.Contains(t, err.Error(), "offset 0")
		require.Equal(t, int64(0), w.Offset())
	})

	t.Run("partial write records bytes written without advancing offset", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 2, io.ErrClosedPipe
		})}

		err := w.WriteFloat(1.5)

		require.Error(t, err)
		require.ErrorIs(t, err, io.ErrClosedPipe)
		require.Contains(t, err.Error(), "offset 0")
		require.Equal(t, int64(0), w.Offset())

		var werr *BinaryWriterError
		require.ErrorAs(t, err, &werr)
		require.Equal(t, int64(2), werr.BytesWritten)
	})

	t.Run("short write wraps ErrShortWrite with pre-write offset and bytes written", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 2, nil
		})}

		err := w.WriteFloat(1.5)

		require.Error(t, err)
		require.ErrorIs(t, err, io.ErrShortWrite)
		require.Contains(t, err.Error(), "offset 0")
		require.Equal(t, int64(0), w.Offset())

		var werr *BinaryWriterError
		require.ErrorAs(t, err, &werr)
		require.Equal(t, int64(2), werr.BytesWritten)
	})

	t.Run("errors.As extracts BinaryWriterError with offset and bytes written", func(t *testing.T) {
		t.Parallel()

		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			return 2, io.ErrClosedPipe
		})}

		err := w.WriteFloat(1.5)

		var werr *BinaryWriterError
		require.ErrorAs(t, err, &werr)
		require.Equal(t, int64(0), werr.Offset)
		require.Equal(t, int64(2), werr.BytesWritten)
		require.ErrorIs(t, werr.Err, io.ErrClosedPipe)
	})

	t.Run("error after successful length prefix reports offset at prefix end", func(t *testing.T) {
		t.Parallel()

		calls := 0
		w := &BinaryWriter{out: writerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				return len(p), nil
			}
			return 1, io.ErrClosedPipe
		})}

		err := w.WriteString("abc")

		require.Error(t, err)
		require.ErrorIs(t, err, io.ErrClosedPipe)

		var werr *BinaryWriterError
		require.ErrorAs(t, err, &werr)
		require.Equal(t, int64(1), werr.Offset)
		require.Equal(t, int64(1), werr.BytesWritten)
		require.Equal(t, int64(1), w.Offset())
	})
}
