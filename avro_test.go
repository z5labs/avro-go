// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"encoding/binary"
	"errors"
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
