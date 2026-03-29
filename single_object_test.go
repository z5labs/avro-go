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

type singleObjectUnmarshalerStub struct {
	fingerprint [8]byte
	unmarshal   func(r *BinaryReader) error
}

func (s singleObjectUnmarshalerStub) Fingerprint() [8]byte {
	return s.fingerprint
}

func (s singleObjectUnmarshalerStub) UnmarshalAvroBinary(r *BinaryReader) error {
	return s.unmarshal(r)
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

func TestUnmarshalSingleObject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
		obj  singleObjectUnmarshalerStub
	}{
		{
			name: "string value",
			data: func() []byte {
				var buf bytes.Buffer
				buf.Write([]byte{0xC3, 0x01})
				var fp [8]byte
				binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"string"`)))
				buf.Write(fp[:])
				buf.Write([]byte{0x0a}) // length 5 zigzag encoded
				buf.WriteString("hello")
				return buf.Bytes()
			}(),
			obj: singleObjectUnmarshalerStub{
				fingerprint: func() [8]byte {
					var fp [8]byte
					binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"string"`)))
					return fp
				}(),
				unmarshal: func(r *BinaryReader) error {
					s, err := r.ReadString()
					if err != nil {
						return err
					}
					if s != "hello" {
						return errors.New("unexpected string value")
					}
					return nil
				},
			},
		},
		{
			name: "int value",
			data: func() []byte {
				var buf bytes.Buffer
				buf.Write([]byte{0xC3, 0x01})
				var fp [8]byte
				binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"int"`)))
				buf.Write(fp[:])
				buf.Write([]byte{0x54}) // 42 zigzag encoded
				return buf.Bytes()
			}(),
			obj: singleObjectUnmarshalerStub{
				fingerprint: func() [8]byte {
					var fp [8]byte
					binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"int"`)))
					return fp
				}(),
				unmarshal: func(r *BinaryReader) error {
					i, err := r.ReadInt()
					if err != nil {
						return err
					}
					if i != 42 {
						return errors.New("unexpected int value")
					}
					return nil
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := UnmarshalSingleObject(bytes.NewReader(tc.data), tc.obj)

			require.NoError(t, err)
		})
	}
}

func TestUnmarshalSingleObject_ReaderError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	unmarshalErr := errors.New("unmarshal failed")

	validMagic := []byte{0xC3, 0x01}
	validFingerprint := func() [8]byte {
		var fp [8]byte
		binary.LittleEndian.PutUint64(fp[:], Fingerprint64([]byte(`"string"`)))
		return fp
	}()

	testCases := []struct {
		name        string
		r           io.Reader
		obj         singleObjectUnmarshalerStub
		expectedErr error
	}{
		{
			name: "error reading magic bytes",
			r:    readerFunc(func(p []byte) (int, error) { return 0, readErr }),
			obj: singleObjectUnmarshalerStub{
				unmarshal: func(r *BinaryReader) error { return nil },
			},
			expectedErr: readErr,
		},
		{
			name: "eof reading magic bytes",
			r:    bytes.NewReader(nil),
			obj: singleObjectUnmarshalerStub{
				unmarshal: func(r *BinaryReader) error { return nil },
			},
			expectedErr: io.EOF,
		},
		{
			name: "bad magic bytes",
			r:    bytes.NewReader([]byte{0x00, 0x00}),
			obj: singleObjectUnmarshalerStub{
				unmarshal: func(r *BinaryReader) error { return nil },
			},
			expectedErr: ErrBadMagic,
		},
		{
			name: "error reading fingerprint",
			r: func() io.Reader {
				calls := 0
				return readerFunc(func(p []byte) (int, error) {
					calls++
					if calls == 1 {
						copy(p, validMagic)
						return len(validMagic), nil
					}
					return 0, readErr
				})
			}(),
			obj: singleObjectUnmarshalerStub{
				unmarshal: func(r *BinaryReader) error { return nil },
			},
			expectedErr: readErr,
		},
		{
			name: "fingerprint mismatch",
			r: func() io.Reader {
				var buf bytes.Buffer
				buf.Write(validMagic)
				buf.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
				return &buf
			}(),
			obj: singleObjectUnmarshalerStub{
				fingerprint: validFingerprint,
				unmarshal:   func(r *BinaryReader) error { return nil },
			},
			expectedErr: ErrFingerprintMismatch,
		},
		{
			name: "error from unmarshal",
			r: func() io.Reader {
				var buf bytes.Buffer
				buf.Write(validMagic)
				buf.Write(validFingerprint[:])
				return &buf
			}(),
			obj: singleObjectUnmarshalerStub{
				fingerprint: validFingerprint,
				unmarshal:   func(r *BinaryReader) error { return unmarshalErr },
			},
			expectedErr: unmarshalErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := UnmarshalSingleObject(tc.r, tc.obj)

			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
