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
	"testing"

	"github.com/stretchr/testify/require"
)

type singleObjectMarshalerStub struct {
	fingerprint [8]byte
	marshal     func(w *BinaryWriter) error
}

func (s singleObjectMarshalerStub) Fingerprint() [8]byte {
	return s.fingerprint
}

func (s singleObjectMarshalerStub) MarshalAvroBinary(w *BinaryWriter) error {
	return s.marshal(w)
}

func TestMarshalSingleObject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		obj      SingleObjectMarshaler
		expected []byte
	}{
		{
			name: "string value",
			obj: singleObjectMarshalerStub{
				fingerprint: func() [8]byte {
					var fp [8]byte
					binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"string"`)))
					return fp
				}(),
				marshal: func(w *BinaryWriter) error {
					return w.WriteString("hello")
				},
			},
			expected: func() []byte {
				var buf bytes.Buffer
				buf.Write([]byte{0xC3, 0x01})
				var fp [8]byte
				binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"string"`)))
				buf.Write(fp[:])
				buf.Write([]byte{0x0a}) // length 5 zigzag encoded
				buf.WriteString("hello")
				return buf.Bytes()
			}(),
		},
		{
			name: "int value",
			obj: singleObjectMarshalerStub{
				fingerprint: func() [8]byte {
					var fp [8]byte
					binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"int"`)))
					return fp
				}(),
				marshal: func(w *BinaryWriter) error {
					return w.WriteInt(42)
				},
			},
			expected: func() []byte {
				var buf bytes.Buffer
				buf.Write([]byte{0xC3, 0x01})
				var fp [8]byte
				binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"int"`)))
				buf.Write(fp[:])
				buf.Write([]byte{0x54}) // 42 zigzag encoded
				return buf.Bytes()
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := MarshalSingleObject(&buf, tc.obj)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestMarshalSingleObject_WriterError(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("write failed")

	testCases := []struct {
		name        string
		w           writerFunc
		obj         SingleObjectMarshaler
		expectedErr error
	}{
		{
			name: "error writing magic bytes",
			w: writerFunc(func(p []byte) (int, error) {
				return 0, writeErr
			}),
			obj: singleObjectMarshalerStub{
				marshal: func(w *BinaryWriter) error {
					return nil
				},
			},
			expectedErr: writeErr,
		},
		{
			name: "short write on magic bytes",
			w: writerFunc(func(p []byte) (int, error) {
				return 1, nil
			}),
			obj: singleObjectMarshalerStub{
				marshal: func(w *BinaryWriter) error {
					return nil
				},
			},
			expectedErr: io.ErrShortWrite,
		},
		{
			name: "error writing fingerprint",
			w: func() writerFunc {
				calls := 0
				return writerFunc(func(p []byte) (int, error) {
					calls++
					if calls == 1 {
						return len(p), nil
					}
					return 0, writeErr
				})
			}(),
			obj: singleObjectMarshalerStub{
				marshal: func(w *BinaryWriter) error {
					return nil
				},
			},
			expectedErr: writeErr,
		},
		{
			name: "short write on fingerprint",
			w: func() writerFunc {
				calls := 0
				return writerFunc(func(p []byte) (int, error) {
					calls++
					if calls == 1 {
						return len(p), nil
					}
					return 4, nil
				})
			}(),
			obj: singleObjectMarshalerStub{
				marshal: func(w *BinaryWriter) error {
					return nil
				},
			},
			expectedErr: io.ErrShortWrite,
		},
		{
			name: "error from marshal",
			w: writerFunc(func(p []byte) (int, error) {
				return len(p), nil
			}),
			obj: singleObjectMarshalerStub{
				marshal: func(w *BinaryWriter) error {
					return writeErr
				},
			},
			expectedErr: writeErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := MarshalSingleObject(tc.w, tc.obj)

			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
