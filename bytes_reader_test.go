// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// requireBytesReaderErr asserts that err is a *BinaryReaderError wrapping
// expected and reported at offset, and that r consumed nothing past offset.
func requireBytesReaderErr(t *testing.T, r *BinaryReader, err error, expected error, offset int64, remaining []byte) {
	t.Helper()

	var rerr *BinaryReaderError
	require.True(t, errors.As(err, &rerr), "expected a *BinaryReaderError, got %T: %v", err, err)
	require.Equal(t, expected, rerr.Err)
	require.Equal(t, offset, rerr.Offset)
	require.Equal(t, offset, r.Offset())
	require.True(t, bytes.Equal(remaining, r.Remaining()), "remaining %x, expected %x", r.Remaining(), remaining)
}

func TestBytesReader(t *testing.T) {
	t.Parallel()

	for _, tc := range validBytesCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded := tc.append(nil)
			trailing := []byte{0x02, 0x04}
			input := append(bytes.Clone(encoded), trailing...)

			r := NewBytesReader(input)
			got, err := tc.read(r)

			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
			require.Equal(t, int64(len(encoded)), r.Offset())
			require.Equal(t, trailing, r.Remaining())
			require.Equal(t, int64(len(input)-len(r.Remaining())), r.Offset())
		})
	}
}

func TestBytesReader_truncated(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name  string
		input []byte
		read  func(r *BinaryReader) (any, error)
	}
	var testCases []testCase
	for _, bc := range validBytesCases() {
		encoded := bc.append(nil)
		for _, prefix := range truncations(encoded) {
			testCases = append(testCases, testCase{
				name:  fmt.Sprintf("%s cut after %d of %d bytes", bc.name, len(prefix), len(encoded)),
				input: prefix,
				read:  bc.read,
			})
		}
	}
	testCases = append(testCases,
		testCase{
			name:  "long with a continuation byte and nothing after it",
			input: []byte{0x80},
			read:  func(r *BinaryReader) (any, error) { return r.ReadLong() },
		},
		testCase{
			name:  "bytes with a length prefix and no payload",
			input: []byte{0x04},
			read:  func(r *BinaryReader) (any, error) { return r.ReadBytes() },
		},
		testCase{
			name:  "string with a length prefix and a short payload",
			input: []byte{0x06, 'a', 'b'},
			read:  func(r *BinaryReader) (any, error) { return r.ReadString() },
		},
		testCase{
			name:  "fixed one byte short",
			input: []byte{0x01, 0x02, 0x03},
			read:  func(r *BinaryReader) (any, error) { return r.ReadFixed(4) },
		},
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// A value read before the truncated one puts its start at a
			// non-zero offset, so the error can be seen to report it.
			input := append([]byte{0x01}, tc.input...)
			r := NewBytesReader(input)
			_, err := r.ReadBool()
			require.NoError(t, err)

			_, err = tc.read(r)

			requireBytesReaderErr(t, r, err, truncationErr(len(tc.input)), 1, tc.input)
		})
	}
}

func TestBytesReader_corrupt(t *testing.T) {
	t.Parallel()

	tenContinuationBytes := bytes.Repeat([]byte{0x80}, 10)
	tenthByte := func(b byte) []byte {
		return append(bytes.Repeat([]byte{0xff}, 9), b)
	}
	readLong := func(r *BinaryReader) error { _, err := r.ReadLong(); return err }
	readInt := func(r *BinaryReader) error { _, err := r.ReadInt(); return err }
	readIndex := func(r *BinaryReader) error { _, err := r.ReadIndex(3); return err }
	readBytes := func(r *BinaryReader) error { _, err := r.ReadBytes(); return err }
	readString := func(r *BinaryReader) error { _, err := r.ReadString(); return err }

	testCases := []struct {
		name        string
		input       []byte
		read        func(r *BinaryReader) error
		expectedErr error
	}{
		{name: "long with a 10th byte of 0x02", input: tenthByte(0x02), read: readLong, expectedErr: ErrOverflow},
		{name: "long with 10 continuation bytes", input: tenContinuationBytes, read: readLong, expectedErr: ErrOverflow},
		{name: "long with 11 bytes", input: append(bytes.Clone(tenContinuationBytes), 0x01), read: readLong, expectedErr: ErrOverflow},
		{name: "int above max int32", input: AppendLong(nil, math.MaxInt32+1), read: readInt, expectedErr: ErrOverflow},
		{name: "int below min int32", input: AppendLong(nil, math.MinInt32-1), read: readInt, expectedErr: ErrOverflow},
		{name: "int whose long overflows", input: tenthByte(0x02), read: readInt, expectedErr: ErrOverflow},
		{name: "index that is negative", input: AppendLong(nil, -1), read: readIndex, expectedErr: ErrInvalidIndex},
		{name: "index equal to count", input: AppendLong(nil, 3), read: readIndex, expectedErr: ErrInvalidIndex},
		{name: "index above count", input: AppendLong(nil, math.MaxInt64), read: readIndex, expectedErr: ErrInvalidIndex},
		{name: "index whose long overflows", input: tenthByte(0x02), read: readIndex, expectedErr: ErrOverflow},
		{name: "bytes with a negative length", input: []byte{0x01, 0xaa}, read: readBytes, expectedErr: ErrNegativeLength},
		{name: "bytes with a length above max int32", input: AppendLong(nil, math.MaxInt32+1), read: readBytes, expectedErr: ErrOverflow},
		{name: "bytes whose length overflows", input: tenthByte(0x02), read: readBytes, expectedErr: ErrOverflow},
		{name: "string with a negative length", input: []byte{0x01, 'a'}, read: readString, expectedErr: ErrNegativeLength},
		{name: "string whose length overflows", input: tenthByte(0x02), read: readString, expectedErr: ErrOverflow},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := append([]byte{0x01}, tc.input...)
			r := NewBytesReader(input)
			_, err := r.ReadBool()
			require.NoError(t, err)

			err = tc.read(r)

			requireBytesReaderErr(t, r, err, tc.expectedErr, 1, tc.input)
		})
	}
}

func TestBytesReader_offset(t *testing.T) {
	t.Parallel()

	t.Run("offset is the input length less the remaining input after every read", func(t *testing.T) {
		t.Parallel()

		input := AppendString(AppendDouble(AppendLong(AppendBool(nil, true), math.MinInt64), math.Pi), "avro")
		r := NewBytesReader(input)
		reads := []func() error{
			func() error { _, err := r.ReadBool(); return err },
			func() error { _, err := r.ReadLong(); return err },
			func() error { _, err := r.ReadDouble(); return err },
			func() error { _, err := r.ReadString(); return err },
		}
		for _, read := range reads {
			require.NoError(t, read())
			require.Equal(t, int64(len(input)-len(r.Remaining())), r.Offset())
		}
		require.Empty(t, r.Remaining())

		_, err := r.ReadLong()
		requireBytesReaderErr(t, r, err, io.EOF, int64(len(input)), nil)
	})

	t.Run("reset bytes starts the offset again from zero", func(t *testing.T) {
		t.Parallel()

		r := NewBytesReader(AppendLong(nil, 64))
		_, err := r.ReadLong()
		require.NoError(t, err)
		require.Equal(t, int64(2), r.Offset())

		next := AppendLong(AppendBool(nil, true), 1)
		r.ResetBytes(next)
		require.Zero(t, r.Offset())
		require.Equal(t, next, r.Remaining())

		v, err := r.ReadBool()
		require.NoError(t, err)
		require.True(t, v)
		require.Equal(t, int64(1), r.Offset())
	})

	t.Run("reset bytes switches an io reader into bytes mode", func(t *testing.T) {
		t.Parallel()

		r := NewBinaryReader(bytes.NewReader(AppendLong(nil, 64)))
		_, err := r.ReadLong()
		require.NoError(t, err)
		require.Nil(t, r.Remaining())

		next := AppendLong(nil, -3)
		r.ResetBytes(next)
		require.Zero(t, r.Offset())
		require.Equal(t, next, r.Remaining())

		v, err := r.ReadLong()
		require.NoError(t, err)
		require.Equal(t, int64(-3), v)
		require.Empty(t, r.Remaining())
	})

	t.Run("an io reader has no remaining input", func(t *testing.T) {
		t.Parallel()

		r := NewBinaryReader(bytes.NewReader([]byte{0x01, 0x02}))
		require.Nil(t, r.Remaining())
		_, err := r.ReadBool()
		require.NoError(t, err)
		require.Nil(t, r.Remaining())
	})

	t.Run("reading from empty input is a clean eof", func(t *testing.T) {
		t.Parallel()

		for _, input := range [][]byte{nil, {}} {
			r := NewBytesReader(input)
			_, err := r.ReadDouble()
			requireBytesReaderErr(t, r, err, io.EOF, 0, nil)
		}
	})
}

func TestBytesReader_discard(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		n           int64
		expectedErr error
		remaining   []byte
	}{
		{name: "discards part of the input", n: 2, remaining: []byte{0x03}},
		{name: "discards all of the input", n: 3, remaining: []byte{}},
		{name: "discards nothing for zero", n: 0, remaining: []byte{0x01, 0x02, 0x03}},
		{name: "discards nothing for a negative count, as from an io reader", n: -1, remaining: []byte{0x01, 0x02, 0x03}},
		{name: "past the end is an unexpected eof", n: 4, expectedErr: io.ErrUnexpectedEOF, remaining: []byte{0x01, 0x02, 0x03}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := []byte{0x01, 0x02, 0x03}
			r := NewBytesReader(input)
			err := r.discard(tc.n)

			if tc.expectedErr != nil {
				requireBytesReaderErr(t, r, err, tc.expectedErr, 0, tc.remaining)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.remaining, r.Remaining())
			require.Equal(t, int64(len(input)-len(tc.remaining)), r.Offset())
		})
	}
}

func TestBytesReader_aliasing(t *testing.T) {
	t.Parallel()

	t.Run("read bytes returns a slice of the input with capacity equal to its length", func(t *testing.T) {
		t.Parallel()

		input := append(AppendBytes(nil, []byte{0xaa, 0xbb, 0xcc}), 0x99)
		r := NewBytesReader(input)

		got, err := r.ReadBytes()
		require.NoError(t, err)
		require.Equal(t, []byte{0xaa, 0xbb, 0xcc}, got)
		require.Equal(t, len(got), cap(got))
		require.Same(t, &input[1], &got[0])

		// Appending must copy rather than overwrite the byte after the value.
		_ = append(got, 0x00)
		require.Equal(t, byte(0x99), input[4])
	})

	t.Run("read fixed returns a slice of the input with capacity equal to its length", func(t *testing.T) {
		t.Parallel()

		input := []byte{0x01, 0x02, 0x03, 0x99}
		r := NewBytesReader(input)

		got, err := r.ReadFixed(3)
		require.NoError(t, err)
		require.Equal(t, []byte{0x01, 0x02, 0x03}, got)
		require.Equal(t, len(got), cap(got))
		require.Same(t, &input[0], &got[0])

		_ = append(got, 0x00)
		require.Equal(t, byte(0x99), input[3])
	})

	t.Run("empty bytes is a non-nil zero-length slice", func(t *testing.T) {
		t.Parallel()

		got, err := NewBytesReader(AppendBytes(nil, nil)).ReadBytes()
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("fixed of size zero is a non-nil zero-length slice even from nil input", func(t *testing.T) {
		t.Parallel()

		r := NewBytesReader(nil)
		got, err := r.ReadFixed(0)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
		require.Zero(t, r.Offset())
	})

	t.Run("read string copies the input", func(t *testing.T) {
		t.Parallel()

		input := AppendString(nil, "avro")
		r := NewBytesReader(input)

		got, err := r.ReadString()
		require.NoError(t, err)

		input[1] = 'X'
		require.Equal(t, "avro", got)
	})
}

func TestBytesReader_arrayReader(t *testing.T) {
	t.Parallel()

	sized := func(items ...int64) []byte {
		return concat(longs(-int64(len(items))), longs(int64(len(longs(items...)))), longs(items...))
	}

	t.Run("decodes items from unsized and sized blocks", func(t *testing.T) {
		t.Parallel()

		data := concat(longs(2), longs(1, math.MinInt64), sized(3, 4), longs(0))
		r := NewBytesReader(data)

		items, err := readAll(NewArrayReader(r))

		require.NoError(t, err)
		require.Equal(t, []int64{1, math.MinInt64, 3, 4}, items)
		require.Equal(t, int64(len(data)), r.Offset())
		require.Empty(t, r.Remaining())
	})

	t.Run("skips sized blocks", func(t *testing.T) {
		t.Parallel()

		data := concat(sized(1, 2, 3), longs(1), longs(4), sized(5), longs(0))
		r := NewBytesReader(data)
		a := NewArrayReader(r)

		skipped, err := a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipSized, skipped)
		require.Equal(t, int64(len(sized(1, 2, 3))), r.Offset())

		var item arrayItem
		ok, err := a.Next(&item)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, arrayItem(4), item)

		skipped, err = a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipSized, skipped)

		skipped, err = a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipNone, skipped)
		require.Empty(t, r.Remaining())
	})

	t.Run("a sized block cut short is an unexpected eof", func(t *testing.T) {
		t.Parallel()

		data := concat(longs(-3), longs(16), longs(1, 2, 3))
		a := NewArrayReader(NewBytesReader(data))

		skipped, err := a.SkipBlock()
		require.Equal(t, SkipNone, skipped)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	testCases := []struct {
		name     string
		data     []byte
		expected error
		items    []int64
	}{
		{name: "empty input", data: nil, expected: ErrTruncatedArray},
		{name: "unterminated array", data: concat(longs(3), longs(1, 2, 3)), expected: ErrTruncatedArray, items: []int64{1, 2, 3}},
		{name: "unterminated after a sized block", data: sized(1, 2), expected: ErrTruncatedArray, items: []int64{1, 2}},
		{name: "count cut mid varint", data: []byte{0x80}, expected: io.ErrUnexpectedEOF},
		{name: "missing block size", data: longs(-1), expected: io.ErrUnexpectedEOF},
		// The item's own read starts at the end of the input, so it reports a
		// clean EOF, exactly as it does from an io.Reader.
		{name: "item cut short", data: concat(longs(2), longs(1)), expected: io.EOF, items: []int64{1}},
		{name: "negative block size", data: concat(longs(-1), longs(-1), longs(1), longs(0)), expected: ErrNegativeLength},
		{name: "block declares more bytes than its items consume", data: concat(longs(-1), longs(2), longs(1), longs(0)), expected: ErrBlockSizeMismatch},
		{name: "block size overflows the reader offset", data: concat(longs(-1), longs(math.MaxInt64)), expected: ErrOverflow},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := NewArrayReader(NewBytesReader(tc.data))
			items, err := readAll(a)

			require.ErrorIs(t, err, tc.expected)
			require.Equal(t, tc.items, items)

			var item arrayItem
			ok, again := a.Next(&item)
			require.False(t, ok)
			require.Equal(t, err, again)
		})
	}
}

// generatedRecord is shaped like the code avroc-gen-go generates for a record
// of a fixed, an enum and two bytes fields.
type generatedRecord struct {
	ID      [16]byte
	Kind    generatedEnum
	Key     []byte
	Payload []byte
}

type generatedEnum int

func (r *generatedRecord) MarshalAvroBinary(w *BinaryWriter) error {
	if err := w.WriteFixed(r.ID[:]); err != nil {
		return err
	}
	if err := w.WriteLong(int64(r.Kind)); err != nil {
		return err
	}
	if err := w.WriteBytes(r.Key); err != nil {
		return err
	}
	return w.WriteBytes(r.Payload)
}

func (r *generatedRecord) UnmarshalAvroBinary(br *BinaryReader) error {
	id, err := br.ReadFixed(16)
	if err != nil {
		return err
	}
	copy(r.ID[:], id)
	kind, err := br.ReadIndex(3)
	if err != nil {
		return err
	}
	r.Kind = generatedEnum(kind)
	if r.Key, err = br.ReadBytes(); err != nil {
		return err
	}
	r.Payload, err = br.ReadBytes()
	return err
}

func encodeGeneratedRecord(t testing.TB) []byte {
	t.Helper()

	rec := generatedRecord{
		ID:      [16]byte{0: 0x01, 15: 0xff},
		Kind:    2,
		Key:     []byte("user-42"),
		Payload: bytes.Repeat([]byte{0xab}, 64),
	}
	var buf bytes.Buffer
	require.NoError(t, MarshalBinary(&buf, &rec))
	return buf.Bytes()
}

func TestUnmarshalBytes(t *testing.T) {
	t.Parallel()

	encoded := encodeGeneratedRecord(t)

	var got generatedRecord
	require.NoError(t, UnmarshalBytes(encoded, &got))

	var want generatedRecord
	require.NoError(t, UnmarshalBinary(bytes.NewReader(encoded), &want))
	require.Equal(t, want, got)
}

var (
	readerSinkBool   bool
	readerSinkInt    int32
	readerSinkLong   int64
	readerSinkIndex  int
	readerSinkF32    float32
	readerSinkF64    float64
	readerSinkBytes  []byte
	readerSinkString string
	readerSinkErr    error
)

func TestBytesReaderAllocations(t *testing.T) {
	// Not parallel: allocation counts are only meaningful when nothing else in
	// this test binary is allocating at the same time.

	long := AppendLong(nil, math.MinInt64)
	index := AppendIndex(nil, 200)
	payload := AppendBytes(nil, bytes.Repeat([]byte{0xab}, 64))
	double := AppendDouble(nil, math.Pi)
	array := concat(longs(-2), longs(int64(len(longs(1, 2)))), longs(1, 2), longs(0))
	r := NewBytesReader(nil)

	testCases := []struct {
		name  string
		input []byte
		read  func()
	}{
		{name: "ReadBool", input: long, read: func() { readerSinkBool, readerSinkErr = r.ReadBool() }},
		{name: "ReadInt", input: index, read: func() { readerSinkInt, readerSinkErr = r.ReadInt() }},
		{name: "ReadLong", input: long, read: func() { readerSinkLong, readerSinkErr = r.ReadLong() }},
		{name: "ReadIndex", input: index, read: func() { readerSinkIndex, readerSinkErr = r.ReadIndex(201) }},
		{name: "ReadFloat", input: double, read: func() { readerSinkF32, readerSinkErr = r.ReadFloat() }},
		{name: "ReadDouble", input: double, read: func() { readerSinkF64, readerSinkErr = r.ReadDouble() }},
		{name: "ReadBytes", input: payload, read: func() { readerSinkBytes, readerSinkErr = r.ReadBytes() }},
		{name: "ReadBytes empty", input: []byte{0x00}, read: func() { readerSinkBytes, readerSinkErr = r.ReadBytes() }},
		{name: "ReadFixed", input: payload, read: func() { readerSinkBytes, readerSinkErr = r.ReadFixed(16) }},
		{name: "ReadFixed size zero", input: nil, read: func() { readerSinkBytes, readerSinkErr = r.ReadFixed(0) }},
		{name: "ArrayReader SkipBlock", input: array, read: func() {
			a := ArrayReader{r: r}
			_, readerSinkErr = a.SkipBlock()
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(100, func() {
				r.ResetBytes(tc.input)
				tc.read()
			})
			require.NoError(t, readerSinkErr)
			require.Zero(t, allocs)
		})
	}

	t.Run("ReadString copies into one allocation", func(t *testing.T) {
		input := AppendString(nil, "hello, avro")
		allocs := testing.AllocsPerRun(100, func() {
			r.ResetBytes(input)
			readerSinkString, readerSinkErr = r.ReadString()
		})
		require.NoError(t, readerSinkErr)
		require.Equal(t, float64(1), allocs)
	})

	t.Run("generated record through a reused reader", func(t *testing.T) {
		encoded := encodeGeneratedRecord(t)
		var rec generatedRecord
		allocs := testing.AllocsPerRun(100, func() {
			r.ResetBytes(encoded)
			readerSinkErr = rec.UnmarshalAvroBinary(r)
		})
		require.NoError(t, readerSinkErr)
		require.Zero(t, allocs)
		require.Empty(t, r.Remaining())
	})
}

// readerBenchCase is one read method and an encoding it succeeds on.
type readerBenchCase struct {
	name    string
	encoded []byte
	read    func(r *BinaryReader) error
}

func readerBenchCases() []readerBenchCase {
	ignore := func(_ any, err error) error { return err }
	return []readerBenchCase{
		{name: "ReadBool", encoded: AppendBool(nil, true), read: func(r *BinaryReader) error { return ignore(r.ReadBool()) }},
		{name: "ReadInt", encoded: AppendInt(nil, math.MaxInt32), read: func(r *BinaryReader) error { return ignore(r.ReadInt()) }},
		{name: "ReadLong", encoded: AppendLong(nil, math.MinInt64), read: func(r *BinaryReader) error { return ignore(r.ReadLong()) }},
		{name: "ReadIndex", encoded: AppendIndex(nil, 200), read: func(r *BinaryReader) error { return ignore(r.ReadIndex(201)) }},
		{name: "ReadFloat", encoded: AppendFloat(nil, math.E), read: func(r *BinaryReader) error { return ignore(r.ReadFloat()) }},
		{name: "ReadDouble", encoded: AppendDouble(nil, math.Pi), read: func(r *BinaryReader) error { return ignore(r.ReadDouble()) }},
		{name: "ReadBytes", encoded: AppendBytes(nil, bytes.Repeat([]byte{0xab}, 64)), read: func(r *BinaryReader) error { return ignore(r.ReadBytes()) }},
		{name: "ReadString", encoded: AppendString(nil, "hello, avro"), read: func(r *BinaryReader) error { return ignore(r.ReadString()) }},
		{name: "ReadFixed", encoded: bytes.Repeat([]byte{0x5a}, 16), read: func(r *BinaryReader) error { return ignore(r.ReadFixed(16)) }},
	}
}

func BenchmarkBinaryReader_ioReader(b *testing.B) {
	for _, bc := range readerBenchCases() {
		b.Run(bc.name, func(b *testing.B) {
			br := bytes.NewReader(bc.encoded)
			r := NewBinaryReader(br)
			for b.Loop() {
				br.Reset(bc.encoded)
				readerSinkErr = bc.read(r)
			}
			if readerSinkErr != nil {
				b.Fatal(readerSinkErr)
			}
		})
	}
}

func BenchmarkBinaryReader_bytes(b *testing.B) {
	for _, bc := range readerBenchCases() {
		b.Run(bc.name, func(b *testing.B) {
			r := NewBytesReader(nil)
			for b.Loop() {
				r.ResetBytes(bc.encoded)
				readerSinkErr = bc.read(r)
			}
			if readerSinkErr != nil {
				b.Fatal(readerSinkErr)
			}
		})
	}
}

func BenchmarkUnmarshalAvroBinary(b *testing.B) {
	encoded := encodeGeneratedRecord(b)

	b.Run("io reader", func(b *testing.B) {
		var rec generatedRecord
		for b.Loop() {
			readerSinkErr = UnmarshalBinary(bytes.NewReader(encoded), &rec)
		}
		if readerSinkErr != nil {
			b.Fatal(readerSinkErr)
		}
	})

	b.Run("UnmarshalBytes", func(b *testing.B) {
		var rec generatedRecord
		for b.Loop() {
			readerSinkErr = UnmarshalBytes(encoded, &rec)
		}
		if readerSinkErr != nil {
			b.Fatal(readerSinkErr)
		}
	})

	b.Run("reused bytes reader", func(b *testing.B) {
		var rec generatedRecord
		r := NewBytesReader(nil)
		for b.Loop() {
			r.ResetBytes(encoded)
			readerSinkErr = rec.UnmarshalAvroBinary(r)
		}
		if readerSinkErr != nil {
			b.Fatal(readerSinkErr)
		}
	})
}
