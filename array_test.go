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

// arrayItem is a minimal Avro long, used as the item type throughout these
// tests. It marshals to a single zigzag varint.
type arrayItem int64

func (v *arrayItem) MarshalAvroBinary(w *BinaryWriter) error {
	return w.WriteLong(int64(*v))
}

func (v *arrayItem) UnmarshalAvroBinary(r *BinaryReader) error {
	l, err := r.ReadLong()
	if err != nil {
		return err
	}
	*v = arrayItem(l)
	return nil
}

// longs encodes each value as a zigzag varint, for building array framing in
// test fixtures.
func longs(vals ...int64) []byte {
	var buf bytes.Buffer
	w := NewBinaryWriter(&buf)
	for _, v := range vals {
		if err := w.WriteLong(v); err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}

func concat(chunks ...[]byte) []byte {
	var buf bytes.Buffer
	for _, c := range chunks {
		buf.Write(c)
	}
	return buf.Bytes()
}

// readAll drains an array, collecting every item it yields.
func readAll(a *ArrayReader) ([]int64, error) {
	var items []int64
	var item arrayItem
	for {
		ok, err := a.Next(&item)
		if err != nil {
			return items, err
		}
		if !ok {
			return items, nil
		}
		items = append(items, int64(item))
	}
}

func TestArrayReader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		data     []byte
		expected []int64
	}{
		{
			name:     "empty array",
			data:     []byte{0x00},
			expected: nil,
		},
		{
			name:     "single unsized block",
			data:     concat(longs(3), longs(1, 2, 3), longs(0)),
			expected: []int64{1, 2, 3},
		},
		{
			name:     "multiple unsized blocks",
			data:     concat(longs(2), longs(1, 2), longs(1), longs(3), longs(0)),
			expected: []int64{1, 2, 3},
		},
		{
			name: "single sized block",
			// count -3 means three items, followed by the block's size in bytes.
			data:     concat(longs(-3), longs(int64(len(longs(1, 2, 3)))), longs(1, 2, 3), longs(0)),
			expected: []int64{1, 2, 3},
		},
		{
			name: "sized and unsized blocks interleaved",
			data: concat(
				longs(-2), longs(int64(len(longs(1, 2)))), longs(1, 2),
				longs(1), longs(3),
				longs(-1), longs(int64(len(longs(4)))), longs(4),
				longs(0),
			),
			expected: []int64{1, 2, 3, 4},
		},
		{
			name:     "multi byte items",
			data:     concat(longs(2), longs(math.MaxInt64, math.MinInt64), longs(0)),
			expected: []int64{math.MaxInt64, math.MinInt64},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewBinaryReader(bytes.NewReader(tc.data))
			items, err := readAll(NewArrayReader(r))

			require.NoError(t, err)
			require.Equal(t, tc.expected, items)
			require.Equal(t, int64(len(tc.data)), r.Offset())
		})
	}
}

func TestArrayReader_exhausted(t *testing.T) {
	t.Parallel()

	a := NewArrayReader(NewBinaryReader(bytes.NewReader(concat(longs(1), longs(7), longs(0)))))

	var item arrayItem
	ok, err := a.Next(&item)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, arrayItem(7), item)

	// Every call past the terminating block keeps reporting the end of the array.
	for range 3 {
		ok, err = a.Next(&item)
		require.NoError(t, err)
		require.False(t, ok)
	}

	skipped, err := a.SkipBlock()
	require.NoError(t, err)
	require.Equal(t, SkipNone, skipped)
}

func TestArrayReader_error(t *testing.T) {
	t.Parallel()

	// A count of math.MinInt64 cannot be negated into an item count.
	minInt64Count := func() []byte {
		var buf [binary.MaxVarintLen64]byte
		n := binary.PutVarint(buf[:], math.MinInt64)
		return buf[:n]
	}()

	testCases := []struct {
		name     string
		data     []byte
		expected error
		items    []int64
	}{
		{
			name:     "empty stream",
			data:     nil,
			expected: ErrTruncatedArray,
		},
		{
			name:     "unterminated array",
			data:     concat(longs(3), longs(1, 2, 3)),
			expected: ErrTruncatedArray,
			items:    []int64{1, 2, 3},
		},
		{
			name:     "count cut mid varint",
			data:     []byte{0x80},
			expected: io.ErrUnexpectedEOF,
		},
		{
			name:     "block size cut mid varint",
			data:     concat(longs(-1), []byte{0x80}),
			expected: io.ErrUnexpectedEOF,
		},
		{
			name:     "missing block size",
			data:     longs(-1),
			expected: io.ErrUnexpectedEOF,
		},
		{
			name:     "item cut short",
			data:     concat(longs(2), longs(1)),
			expected: io.EOF,
			items:    []int64{1},
		},
		{
			name:     "negative block size",
			data:     concat(longs(-1), longs(-1), longs(1), longs(0)),
			expected: ErrNegativeLength,
		},
		{
			name:     "count overflows on negation",
			data:     minInt64Count,
			expected: ErrOverflow,
		},
		{
			name: "block declares more bytes than its items consume",
			// One item encoded in one byte, but the block claims two.
			data:     concat(longs(-1), longs(2), longs(1), longs(0)),
			expected: ErrBlockSizeMismatch,
		},
		{
			name:     "block size overflows the reader offset",
			data:     concat(longs(-1), longs(math.MaxInt64)),
			expected: ErrOverflow,
		},
		{
			name:     "count runs past the maximum varint length",
			data:     bytes.Repeat([]byte{0x80}, binary.MaxVarintLen64),
			expected: ErrOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := NewArrayReader(NewBinaryReader(bytes.NewReader(tc.data)))
			items, err := readAll(a)

			require.ErrorIs(t, err, tc.expected)
			require.Equal(t, tc.items, items)

			// The error is sticky: it is reported again rather than the reader
			// silently resuming past it.
			var item arrayItem
			ok, again := a.Next(&item)
			require.False(t, ok)
			require.Equal(t, err, again)

			_, again = a.SkipBlock()
			require.Equal(t, err, again)
		})
	}
}

func TestArrayReader_itemError(t *testing.T) {
	t.Parallel()

	errItem := errors.New("item")
	failing := binaryUnmarshalerFunc(func(r *BinaryReader) error {
		return errItem
	})

	a := NewArrayReader(NewBinaryReader(bytes.NewReader(concat(longs(1), longs(1), longs(0)))))

	ok, err := a.Next(failing)
	require.False(t, ok)
	require.ErrorIs(t, err, errItem)
}

func TestArrayReaderSkipBlock(t *testing.T) {
	t.Parallel()

	t.Run("discards a sized block without decoding any item", func(t *testing.T) {
		t.Parallel()

		data := concat(
			longs(-3), longs(int64(len(longs(1, 2, 3)))), longs(1, 2, 3),
			longs(1), longs(4),
			longs(0),
		)
		r := NewBinaryReader(bytes.NewReader(data))
		a := NewArrayReader(r)

		skipped, err := a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipSized, skipped)
		require.Equal(t, int64(len(data)-len(longs(1))-len(longs(4))-len(longs(0))), r.Offset())

		items, err := readAll(a)
		require.NoError(t, err)
		require.Equal(t, []int64{4}, items)
	})

	t.Run("discards the remainder of a partly decoded sized block", func(t *testing.T) {
		t.Parallel()

		data := concat(
			longs(-3), longs(int64(len(longs(1, 2, 3)))), longs(1, 2, 3),
			longs(1), longs(4),
			longs(0),
		)
		a := NewArrayReader(NewBinaryReader(bytes.NewReader(data)))

		var item arrayItem
		ok, err := a.Next(&item)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, arrayItem(1), item)

		skipped, err := a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipSized, skipped)

		items, err := readAll(a)
		require.NoError(t, err)
		require.Equal(t, []int64{4}, items)
	})

	t.Run("consumes nothing when the block is unsized", func(t *testing.T) {
		t.Parallel()

		data := concat(longs(3), longs(1, 2, 3), longs(0))
		r := NewBinaryReader(bytes.NewReader(data))
		a := NewArrayReader(r)

		skipped, err := a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipUnsized, skipped)

		// Only the block header was read; the items are still there to drain.
		require.Equal(t, int64(len(longs(3))), r.Offset())

		items, err := readAll(a)
		require.NoError(t, err)
		require.Equal(t, []int64{1, 2, 3}, items)
	})

	t.Run("reports the unsized path repeatedly", func(t *testing.T) {
		t.Parallel()

		a := NewArrayReader(NewBinaryReader(bytes.NewReader(concat(longs(1), longs(1), longs(0)))))

		for range 3 {
			skipped, err := a.SkipBlock()
			require.NoError(t, err)
			require.Equal(t, SkipUnsized, skipped)
		}
	})

	t.Run("reports no block on an empty array", func(t *testing.T) {
		t.Parallel()

		a := NewArrayReader(NewBinaryReader(bytes.NewReader(longs(0))))

		skipped, err := a.SkipBlock()
		require.NoError(t, err)
		require.Equal(t, SkipNone, skipped)
	})

	t.Run("errors when the block declares fewer bytes than its items consume", func(t *testing.T) {
		t.Parallel()

		// Two items, but the block claims to be empty.
		data := concat(longs(-2), longs(0), longs(1, 2), longs(0))
		a := NewArrayReader(NewBinaryReader(bytes.NewReader(data)))

		var item arrayItem
		ok, err := a.Next(&item)
		require.NoError(t, err)
		require.True(t, ok)

		skipped, err := a.SkipBlock()
		require.Equal(t, SkipNone, skipped)
		require.ErrorIs(t, err, ErrBlockSizeMismatch)
	})

	t.Run("errors when a sized block is cut short", func(t *testing.T) {
		t.Parallel()

		// The block claims 16 bytes but only three follow.
		data := concat(longs(-3), longs(16), longs(1, 2, 3))
		a := NewArrayReader(NewBinaryReader(bytes.NewReader(data)))

		skipped, err := a.SkipBlock()
		require.Equal(t, SkipNone, skipped)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}

func TestArrayWriter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		opts     []ArrayWriterOption
		items    []int64
		expected []byte
	}{
		{
			name:     "empty array",
			items:    nil,
			expected: []byte{0x00},
		},
		{
			name:  "unsized blocks default to one item per block",
			items: []int64{1, 2, 3},
			expected: []byte{
				0x02, 0x02, // count 1, item 1
				0x02, 0x04, // count 1, item 2
				0x02, 0x06, // count 1, item 3
				0x00, // terminator
			},
		},
		{
			name:  "sized blocks batch until the buffer fills",
			opts:  []ArrayWriterOption{WithSizedBlocks(2)},
			items: []int64{1, 2, 3},
			expected: []byte{
				0x03, 0x04, 0x02, 0x04, // count -2, size 2, items 1 and 2
				0x01, 0x02, 0x06, // count -1, size 1, item 3
				0x00, // terminator
			},
		},
		{
			name:  "sized blocks hold every item when the buffer never fills",
			opts:  []ArrayWriterOption{WithSizedBlocks(1024)},
			items: []int64{1, 2, 3},
			expected: []byte{
				0x05, 0x06, 0x02, 0x04, 0x06, // count -3, size 3, items 1, 2 and 3
				0x00, // terminator
			},
		},
		{
			name:     "sized blocks on an empty array write only the terminator",
			opts:     []ArrayWriterOption{WithSizedBlocks(1024)},
			items:    nil,
			expected: []byte{0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			a := NewArrayWriter(NewBinaryWriter(&buf), tc.opts...)

			for _, v := range tc.items {
				item := arrayItem(v)
				require.NoError(t, a.Write(&item))
			}
			require.NoError(t, a.Close())

			require.Equal(t, tc.expected, buf.Bytes())
		})
	}
}

func TestArrayWriter_singleItemLargerThanBuffer(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	a := NewArrayWriter(NewBinaryWriter(&buf), WithSizedBlocks(1))

	// math.MaxInt64 encodes to ten bytes, well past the one byte buffer, so it
	// is flushed as a block of its own.
	item := arrayItem(math.MaxInt64)
	require.NoError(t, a.Write(&item))
	require.NoError(t, a.Close())

	expected := concat(longs(-1), longs(int64(len(longs(math.MaxInt64)))), longs(math.MaxInt64), longs(0))
	require.Equal(t, expected, buf.Bytes())
}

func TestArrayWriter_close(t *testing.T) {
	t.Parallel()

	t.Run("is idempotent", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		a := NewArrayWriter(NewBinaryWriter(&buf))

		require.NoError(t, a.Close())
		require.NoError(t, a.Close())
		require.Equal(t, []byte{0x00}, buf.Bytes())
	})

	t.Run("rejects writes afterwards", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		a := NewArrayWriter(NewBinaryWriter(&buf))
		require.NoError(t, a.Close())

		item := arrayItem(1)
		require.ErrorIs(t, a.Write(&item), ErrArrayWriterClosed)
		require.Equal(t, []byte{0x00}, buf.Bytes())
	})

	t.Run("omitting it leaves a detectably truncated array", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		a := NewArrayWriter(NewBinaryWriter(&buf))

		for _, v := range []int64{1, 2, 3} {
			item := arrayItem(v)
			require.NoError(t, a.Write(&item))
		}
		// Deliberately not closed.

		items, err := readAll(NewArrayReader(NewBinaryReader(bytes.NewReader(buf.Bytes()))))
		require.ErrorIs(t, err, ErrTruncatedArray)
		require.Equal(t, []int64{1, 2, 3}, items)
	})
}

func TestArrayWriter_error(t *testing.T) {
	t.Parallel()

	errWrite := errors.New("write")

	for _, tc := range []struct {
		name string
		opts []ArrayWriterOption
	}{
		{name: "unsized blocks"},
		{name: "sized blocks", opts: []ArrayWriterOption{WithSizedBlocks(1024)}},
	} {
		t.Run("propagates and sticks on a failed item with "+tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			a := NewArrayWriter(NewBinaryWriter(&buf), tc.opts...)

			failing := binaryMarshalerFunc(func(w *BinaryWriter) error {
				return errWrite
			})

			require.ErrorIs(t, a.Write(failing), errWrite)

			item := arrayItem(1)
			require.ErrorIs(t, a.Write(&item), errWrite)
			require.ErrorIs(t, a.Close(), errWrite)
		})
	}

	t.Run("propagates a failed block count on an unsized write", func(t *testing.T) {
		t.Parallel()

		a := NewArrayWriter(NewBinaryWriter(failAfter(0, errWrite)))

		item := arrayItem(1)
		require.ErrorIs(t, a.Write(&item), errWrite)
	})

	// A sized block is flushed as a count, then a size, then the buffered
	// items, so each of the three can fail on its own.
	for i, name := range []string{"count", "size", "items"} {
		t.Run("propagates a failed block "+name+" on close", func(t *testing.T) {
			t.Parallel()

			a := NewArrayWriter(NewBinaryWriter(failAfter(i, errWrite)), WithSizedBlocks(1024))

			item := arrayItem(1)
			require.NoError(t, a.Write(&item))
			require.ErrorIs(t, a.Close(), errWrite)
		})
	}

	t.Run("propagates a failed terminator", func(t *testing.T) {
		t.Parallel()

		out := writerFunc(func(p []byte) (int, error) {
			return 0, errWrite
		})
		a := NewArrayWriter(NewBinaryWriter(out))

		require.ErrorIs(t, a.Close(), errWrite)
	})
}

// failAfter returns a writer that accepts n writes and then fails every one
// thereafter.
func failAfter(n int, err error) io.Writer {
	return writerFunc(func(p []byte) (int, error) {
		if n == 0 {
			return 0, err
		}
		n--
		return len(p), nil
	})
}

func TestWriteArray(t *testing.T) {
	t.Parallel()

	t.Run("terminates the array", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		err := WriteArray(NewBinaryWriter(&buf), func(a *ArrayWriter) error {
			for _, v := range []int64{1, 2, 3} {
				item := arrayItem(v)
				if err := a.Write(&item); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err)

		items, err := readAll(NewArrayReader(NewBinaryReader(bytes.NewReader(buf.Bytes()))))
		require.NoError(t, err)
		require.Equal(t, []int64{1, 2, 3}, items)
	})

	t.Run("leaves the array unterminated when the callback fails", func(t *testing.T) {
		t.Parallel()

		errItems := errors.New("items")

		var buf bytes.Buffer
		err := WriteArray(NewBinaryWriter(&buf), func(a *ArrayWriter) error {
			item := arrayItem(1)
			if err := a.Write(&item); err != nil {
				return err
			}
			return errItems
		})
		require.ErrorIs(t, err, errItems)

		_, err = readAll(NewArrayReader(NewBinaryReader(bytes.NewReader(buf.Bytes()))))
		require.ErrorIs(t, err, ErrTruncatedArray)
	})
}

func TestArrayRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		opts  []ArrayWriterOption
		items []int64
	}{
		{name: "unsized empty", items: nil},
		{name: "unsized single item", items: []int64{42}},
		{name: "unsized many items", items: sequence(1000)},
		{name: "sized empty", opts: []ArrayWriterOption{WithSizedBlocks(64)}, items: nil},
		{name: "sized single item", opts: []ArrayWriterOption{WithSizedBlocks(64)}, items: []int64{42}},
		{name: "sized many small blocks", opts: []ArrayWriterOption{WithSizedBlocks(16)}, items: sequence(1000)},
		{name: "sized one large block", opts: []ArrayWriterOption{WithSizedBlocks(1 << 20)}, items: sequence(1000)},
		{name: "sized default buffer", opts: []ArrayWriterOption{WithSizedBlocks(0)}, items: sequence(1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := WriteArray(NewBinaryWriter(&buf), func(a *ArrayWriter) error {
				for _, v := range tc.items {
					item := arrayItem(v)
					if err := a.Write(&item); err != nil {
						return err
					}
				}
				return nil
			}, tc.opts...)
			require.NoError(t, err)

			items, err := readAll(NewArrayReader(NewBinaryReader(bytes.NewReader(buf.Bytes()))))
			require.NoError(t, err)

			var expected []int64
			if len(tc.items) > 0 {
				expected = tc.items
			}
			require.Equal(t, expected, items)
		})
	}
}

func TestArrayRoundTripSkippingSizedBlocks(t *testing.T) {
	t.Parallel()

	items := sequence(1000)

	var buf bytes.Buffer
	err := WriteArray(NewBinaryWriter(&buf), func(a *ArrayWriter) error {
		for _, v := range items {
			item := arrayItem(v)
			if err := a.Write(&item); err != nil {
				return err
			}
		}
		return nil
	}, WithSizedBlocks(16))
	require.NoError(t, err)

	// Every block is sized, so the whole array can be discarded without a
	// single item being decoded.
	a := NewArrayReader(NewBinaryReader(bytes.NewReader(buf.Bytes())))
	blocks := 0
	for {
		skipped, err := a.SkipBlock()
		require.NoError(t, err)
		require.NotEqual(t, SkipUnsized, skipped)
		if skipped == SkipNone {
			break
		}
		blocks++
	}
	require.Greater(t, blocks, 1)
}

func sequence(n int) []int64 {
	items := make([]int64, n)
	for i := range items {
		items[i] = int64(i)
	}
	return items
}

func TestArrayStreamingAllocations(t *testing.T) {
	// Not parallel: allocation counts are only meaningful when nothing else in
	// this test binary is allocating at the same time.

	const (
		few  = 1_000
		many = 100_000

		// Every item has the same encoded width so that item count is the only
		// thing that differs between the two measurements.
		value = arrayItem(1234)
	)

	build := func(n int, opts ...ArrayWriterOption) []byte {
		var buf bytes.Buffer
		err := WriteArray(NewBinaryWriter(&buf), func(a *ArrayWriter) error {
			item := value
			for range n {
				if err := a.Write(&item); err != nil {
					return err
				}
			}
			return nil
		}, opts...)
		require.NoError(t, err)
		return buf.Bytes()
	}

	readAllocs := func(data []byte) float64 {
		return testing.AllocsPerRun(3, func() {
			a := NewArrayReader(NewBinaryReader(bytes.NewReader(data)))
			var item arrayItem
			for {
				ok, err := a.Next(&item)
				if err != nil || !ok {
					return
				}
			}
		})
	}

	writeAllocs := func(n int, opts ...ArrayWriterOption) float64 {
		return testing.AllocsPerRun(3, func() {
			_ = WriteArray(NewBinaryWriter(io.Discard), func(a *ArrayWriter) error {
				item := value
				for range n {
					if err := a.Write(&item); err != nil {
						return err
					}
				}
				return nil
			}, opts...)
		})
	}

	t.Run("reading an unsized array", func(t *testing.T) {
		require.Equal(t, readAllocs(build(few)), readAllocs(build(many)))
	})

	t.Run("reading a sized array", func(t *testing.T) {
		opt := WithSizedBlocks(64)
		require.Equal(t, readAllocs(build(few, opt)), readAllocs(build(many, opt)))
	})

	t.Run("writing an unsized array", func(t *testing.T) {
		require.Equal(t, writeAllocs(few), writeAllocs(many))
	})

	t.Run("writing a sized array", func(t *testing.T) {
		require.Equal(t, writeAllocs(few, WithSizedBlocks(64)), writeAllocs(many, WithSizedBlocks(64)))
	})
}

func TestSkipString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		skip     Skip
		expected string
	}{
		{name: "none", skip: SkipNone, expected: "none"},
		{name: "sized", skip: SkipSized, expected: "sized"},
		{name: "unsized", skip: SkipUnsized, expected: "unsized"},
		{name: "out of range", skip: Skip(42), expected: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.skip.String())
		})
	}
}
