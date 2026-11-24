// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeToWriter encodes with write through an io.Writer-mode BinaryWriter,
// which is the reference append mode is compared against.
func writeToWriter(t *testing.T, write func(w *BinaryWriter) error) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, write(NewBinaryWriter(&buf)))
	return buf.Bytes()
}

func TestAppendWriter(t *testing.T) {
	t.Parallel()

	for _, tc := range validBytesCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expected := writeToWriter(t, tc.write)

			testCases := []struct {
				name string
				dst  []byte
			}{
				{name: "onto nil", dst: nil},
				{name: "onto an empty buffer with room", dst: make([]byte, 0, 128)},
				{name: "onto a buffer with no room", dst: []byte{}},
				{name: "after a prefix", dst: []byte{0x02, 0x04}},
			}
			for _, dc := range testCases {
				t.Run(dc.name, func(t *testing.T) {
					t.Parallel()

					prefix := bytes.Clone(dc.dst)
					w := NewAppendWriter(dc.dst)

					err := tc.write(w)

					require.NoError(t, err)
					require.Equal(t, append(prefix, expected...), w.Bytes())
					require.Equal(t, int64(len(expected)), w.Offset())
					require.Equal(t, int64(len(w.Bytes())), int64(len(prefix))+w.Offset())
				})
			}
		})
	}
}

func TestAppendWriter_record(t *testing.T) {
	t.Parallel()

	// Every case written one after another, as a record's fields are.
	writeAll := func(w *BinaryWriter) error {
		for _, tc := range validBytesCases() {
			if err := tc.write(w); err != nil {
				return err
			}
		}
		return nil
	}
	expected := writeToWriter(t, writeAll)

	w := NewAppendWriter(nil)
	err := writeAll(w)

	require.NoError(t, err)
	require.Equal(t, expected, w.Bytes())
	require.Equal(t, int64(len(expected)), w.Offset())
}

func TestAppendWriter_marshaler(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		v    BinaryMarshaler
	}{
		{name: "message", v: message{content: "hello, avro"}},
		{name: "array item", v: func() *arrayItem { v := arrayItem(-1234); return &v }()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			require.NoError(t, MarshalBinary(&buf, tc.v))

			w := NewAppendWriter(nil)
			err := tc.v.MarshalAvroBinary(w)

			require.NoError(t, err)
			require.Equal(t, buf.Bytes(), w.Bytes())
		})
	}
}

func TestAppendWriter_neverTouchesWriter(t *testing.T) {
	t.Parallel()

	errWrite := errors.New("write")
	touched := 0
	w := NewBinaryWriter(writerFunc(func(p []byte) (int, error) {
		touched++
		return 0, errWrite
	}))
	require.Error(t, w.WriteBool(true))
	require.Equal(t, 1, touched)

	w.ResetAppend(nil)
	for _, tc := range validBytesCases() {
		require.NoError(t, tc.write(w), tc.name)
	}

	require.Equal(t, 1, touched)
}

func TestAppendWriter_offset(t *testing.T) {
	t.Parallel()

	t.Run("counts from len(dst) at NewAppendWriter", func(t *testing.T) {
		t.Parallel()

		w := NewAppendWriter([]byte{0x01, 0x02, 0x03})
		require.Zero(t, w.Offset())
		require.Equal(t, []byte{0x01, 0x02, 0x03}, w.Bytes())

		require.NoError(t, w.WriteLong(64))

		require.Equal(t, int64(2), w.Offset())
		require.Equal(t, []byte{0x01, 0x02, 0x03, 0x80, 0x01}, w.Bytes())
	})

	t.Run("ResetAppend counts from len(dst) again", func(t *testing.T) {
		t.Parallel()

		w := NewAppendWriter(nil)
		require.NoError(t, w.WriteString("abc"))
		require.Equal(t, int64(4), w.Offset())

		w.ResetAppend([]byte{0xff})

		require.Zero(t, w.Offset())
		require.Equal(t, []byte{0xff}, w.Bytes())
		require.NoError(t, w.WriteBool(true))
		require.Equal(t, int64(1), w.Offset())
		require.Equal(t, []byte{0xff, 0x01}, w.Bytes())
	})

	t.Run("ResetAppend discards an io.Writer-mode offset", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		w := NewBinaryWriter(&buf)
		require.NoError(t, w.WriteDouble(math.Pi))
		require.Equal(t, int64(8), w.Offset())

		w.ResetAppend(nil)

		require.Zero(t, w.Offset())
		require.NoError(t, w.WriteInt(-1))
		require.Equal(t, int64(1), w.Offset())
		require.Equal(t, []byte{0x01}, w.Bytes())
		require.Equal(t, 8, buf.Len())
	})
}

func TestAppendWriter_reusesDst(t *testing.T) {
	t.Parallel()

	dst := make([]byte, 1, 64)
	w := NewAppendWriter(dst)
	require.NoError(t, w.WriteString("hello"))

	got := w.Bytes()
	require.Equal(t, []byte{0x00, 0x0a, 'h', 'e', 'l', 'l', 'o'}, got)
	require.Same(t, &dst[0], &got[0], "a dst with room should be appended to in place")
}

func TestBinaryWriter_bytesInWriterMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := NewBinaryWriter(&buf)
	require.Nil(t, w.Bytes())

	require.NoError(t, w.WriteString("abc"))

	require.Nil(t, w.Bytes())
	require.Equal(t, int64(4), w.Offset())
	require.Equal(t, []byte{0x06, 'a', 'b', 'c'}, buf.Bytes())
}

func TestArrayWriter_appendMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		opts  []ArrayWriterOption
		items []int64
	}{
		{name: "empty unsized array", items: nil},
		{name: "unsized blocks", items: []int64{1, -2, 300, math.MaxInt64}},
		{name: "sized blocks that flush", opts: []ArrayWriterOption{WithSizedBlocks(2)}, items: []int64{1, -2, 300, math.MaxInt64}},
		{name: "sized blocks that never fill", opts: []ArrayWriterOption{WithSizedBlocks(1024)}, items: []int64{1, -2, 300}},
		{name: "empty sized array", opts: []ArrayWriterOption{WithSizedBlocks(1024)}, items: nil},
		{name: "many sized blocks", opts: []ArrayWriterOption{WithSizedBlocks(16)}, items: sequence(1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			writeItems := func(w *BinaryWriter) error {
				return WriteArray(w, func(a *ArrayWriter) error {
					for _, v := range tc.items {
						item := arrayItem(v)
						if err := a.Write(&item); err != nil {
							return err
						}
					}
					return nil
				}, tc.opts...)
			}
			expected := writeToWriter(t, writeItems)

			w := NewAppendWriter([]byte{0x7f})
			err := writeItems(w)

			require.NoError(t, err)
			require.Equal(t, append([]byte{0x7f}, expected...), w.Bytes())
			require.Equal(t, int64(len(expected)), w.Offset())

			got, err := readAll(NewArrayReader(NewBytesReader(w.Bytes()[1:])))
			require.NoError(t, err)
			require.Equal(t, tc.items, got)
		})
	}
}

var appendSinkErr error

func TestAppendWriterAllocations(t *testing.T) {
	// Not parallel: allocation counts are only meaningful when nothing else in
	// this test binary is allocating at the same time.

	value := bytes.Repeat([]byte{0xcd}, 64)
	dst := make([]byte, 0, 256)
	w := NewAppendWriter(dst)
	msg := BinaryMarshaler(message{content: "hello, avro"})

	testCases := []struct {
		name  string
		write func() error
	}{
		{name: "WriteBool", write: func() error { return w.WriteBool(true) }},
		{name: "WriteInt", write: func() error { return w.WriteInt(math.MinInt32) }},
		{name: "WriteLong", write: func() error { return w.WriteLong(math.MinInt64) }},
		{name: "WriteFloat", write: func() error { return w.WriteFloat(math.E) }},
		{name: "WriteDouble", write: func() error { return w.WriteDouble(math.Pi) }},
		{name: "WriteBytes", write: func() error { return w.WriteBytes(value) }},
		{name: "WriteFixed", write: func() error { return w.WriteFixed(value) }},
		{name: "WriteString", write: func() error { return w.WriteString("hello, avro") }},
		{name: "MarshalAvroBinary", write: func() error { return msg.MarshalAvroBinary(w) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(100, func() {
				w.ResetAppend(dst[:0])
				appendSinkErr = tc.write()
			})

			require.Zero(t, allocs)
			require.NoError(t, appendSinkErr)
			require.Same(t, &dst[:1][0], &w.Bytes()[0], "the write should have gone into dst")
		})
	}

	arrayCases := []struct {
		name string
		opts []ArrayWriterOption
	}{
		{name: "ArrayWriter with unsized blocks"},
		{name: "ArrayWriter with sized blocks", opts: []ArrayWriterOption{WithSizedBlocks(16)}},
	}
	for _, tc := range arrayCases {
		t.Run(tc.name, func(t *testing.T) {
			// Room for every item AllocsPerRun writes, so that only the
			// ArrayWriter itself could allocate.
			w := NewAppendWriter(make([]byte, 0, 4096))
			a := NewArrayWriter(w, tc.opts...)
			item := arrayItem(1234)

			allocs := testing.AllocsPerRun(100, func() {
				appendSinkErr = a.Write(&item)
			})

			require.Zero(t, allocs)
			require.NoError(t, appendSinkErr)
			require.NoError(t, a.Close())
			require.Equal(t, 4096, cap(w.Bytes()), "the items should have fit without growing dst")
		})
	}
}

func BenchmarkAppendWriter(b *testing.B) {
	dst := make([]byte, 0, 256)
	w := NewAppendWriter(dst)
	value := bytes.Repeat([]byte{0xcd}, 16)

	b.ReportAllocs()
	for b.Loop() {
		w.ResetAppend(dst[:0])
		_ = w.WriteLong(math.MinInt64)
		_ = w.WriteString("hello, avro")
		_ = w.WriteBytes(value)
		_ = w.WriteDouble(math.Pi)
	}
}
