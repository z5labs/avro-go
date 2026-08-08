// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"errors"
	"io"
	"math"
)

var (
	// ErrTruncatedArray is returned when a stream ends before an array's
	// terminating zero-count block. It is what an ArrayWriter that was never
	// closed leaves behind, so callers can distinguish a truncated array from
	// a genuinely empty one.
	ErrTruncatedArray = errors.New("avro: truncated array")

	// ErrBlockSizeMismatch is returned when the items of a sized block do not
	// consume exactly the number of bytes the block declared.
	ErrBlockSizeMismatch = errors.New("avro: array block size mismatch")

	// ErrArrayWriterClosed is returned by ArrayWriter.Write after the writer
	// has been closed.
	ErrArrayWriterClosed = errors.New("avro: array writer closed")
)

// Skip reports how [ArrayReader.SkipBlock] handled the current block.
type Skip int

const (
	// SkipNone reports that no block remained to skip because the array's
	// terminating zero-count block has been reached.
	SkipNone Skip = iota

	// SkipSized reports that the current block declared its encoded size in
	// bytes and so was discarded without decoding any item.
	SkipSized

	// SkipUnsized reports that the current block did not declare its encoded
	// size. Item boundaries inside such a block can only be found by decoding,
	// so SkipBlock consumed nothing and the caller must drain the block with
	// [ArrayReader.Next] instead.
	SkipUnsized
)

func (s Skip) String() string {
	switch s {
	case SkipNone:
		return "none"
	case SkipSized:
		return "sized"
	case SkipUnsized:
		return "unsized"
	}
	return "unknown"
}

// ArrayReader decodes an Avro array one item at a time.
//
// An Avro array is encoded as a series of blocks, each a long item count
// followed by that many items, terminated by a block whose count is zero. A
// negative count is the absolute item count and is followed by a long giving
// the block's encoded size in bytes; see [ArrayReader.SkipBlock].
//
// ArrayReader holds no item state of its own, so memory use is a function of
// the destination passed to [ArrayReader.Next] rather than of the array's
// length.
type ArrayReader struct {
	r *BinaryReader

	remaining int64 // items left to decode in the current block
	blockEnd  int64 // reader offset at which the current block ends; only valid when sized
	sized     bool  // whether the current block declared its encoded size
	done      bool  // whether the terminating zero-count block has been read
	err       error // sticky error
}

// NewArrayReader returns an ArrayReader that decodes an array from r.
func NewArrayReader(r *BinaryReader) *ArrayReader {
	return &ArrayReader{r: r}
}

// Next decodes the next item into v. It reports false at the array's
// terminating zero-count block, and keeps reporting false thereafter.
//
// Passing the same v on every call reuses a single destination for the whole
// array; passing a fresh v each time keeps every item. That choice is the
// caller's.
//
// Once Next returns an error, every later call to Next or
// [ArrayReader.SkipBlock] returns that same error.
func (a *ArrayReader) Next(v BinaryUnmarshaler) (bool, error) {
	if a.err != nil {
		return false, a.err
	}
	if a.done {
		return false, nil
	}
	if a.remaining == 0 {
		ok, err := a.nextBlock()
		if err != nil || !ok {
			return false, err
		}
	}

	if err := v.UnmarshalAvroBinary(a.r); err != nil {
		return false, a.fail(err)
	}
	a.remaining--

	if a.remaining == 0 && a.sized && a.r.Offset() != a.blockEnd {
		return false, a.fail(a.r.wrapErr(ErrBlockSizeMismatch))
	}
	return true, nil
}

// SkipBlock discards the remainder of the current block without decoding it
// and reports which path it took.
//
// It returns [SkipSized] when the block declared its encoded size, in which
// case the remaining bytes are discarded straight from the underlying reader
// and no item is decoded. It returns [SkipUnsized] when the block declared no
// size: nothing is consumed, because finding the block's end would mean
// decoding every remaining item, which is no cheaper than draining the block
// with [ArrayReader.Next]. It returns [SkipNone] once the array's terminating
// zero-count block has been reached.
func (a *ArrayReader) SkipBlock() (Skip, error) {
	if a.err != nil {
		return SkipNone, a.err
	}
	if a.done {
		return SkipNone, nil
	}
	if a.remaining == 0 {
		ok, err := a.nextBlock()
		if err != nil || !ok {
			return SkipNone, err
		}
	}
	if !a.sized {
		return SkipUnsized, nil
	}

	switch n := a.blockEnd - a.r.Offset(); {
	case n < 0:
		return SkipNone, a.fail(a.r.wrapErr(ErrBlockSizeMismatch))
	case n > 0:
		if err := a.r.discard(n); err != nil {
			return SkipNone, a.fail(err)
		}
	}
	a.remaining = 0
	return SkipSized, nil
}

// nextBlock reads the header of the next block. It reports false once the
// array's terminating zero-count block has been read.
func (a *ArrayReader) nextBlock() (bool, error) {
	start := a.r.Offset()
	count, err := a.r.ReadLong()
	if err != nil {
		// A clean EOF before any byte of the count is an array that was never
		// terminated; anywhere else it is a stream cut mid-value.
		if a.r.Offset() == start && isEOF(err) {
			return false, a.fail(a.r.wrapErr(ErrTruncatedArray))
		}
		return false, a.fail(unexpectedEOF(err))
	}
	if count == 0 {
		a.done = true
		return false, nil
	}

	a.sized = count < 0
	if a.sized {
		if count == math.MinInt64 {
			return false, a.fail(a.r.wrapErr(ErrOverflow))
		}
		count = -count

		size, err := a.r.ReadLong()
		if err != nil {
			return false, a.fail(unexpectedEOF(err))
		}
		if size < 0 {
			return false, a.fail(a.r.wrapErr(ErrNegativeLength))
		}
		a.blockEnd = a.r.Offset() + size
		if a.blockEnd < 0 {
			return false, a.fail(a.r.wrapErr(ErrOverflow))
		}
	}

	a.remaining = count
	return true, nil
}

func (a *ArrayReader) fail(err error) error {
	a.err = err
	return err
}

func isEOF(err error) bool {
	var rerr *BinaryReaderError
	return errors.As(err, &rerr) && errors.Is(rerr.Err, io.EOF)
}

// unexpectedEOF rewrites a clean io.EOF into io.ErrUnexpectedEOF, preserving
// the offset it was reported at. Every value an ArrayReader reads for itself is
// required by the array framing, so reaching EOF while reading one is a short
// read rather than a clean end of input.
func unexpectedEOF(err error) error {
	var rerr *BinaryReaderError
	if errors.As(err, &rerr) && errors.Is(rerr.Err, io.EOF) {
		return &BinaryReaderError{Offset: rerr.Offset, Err: io.ErrUnexpectedEOF}
	}
	return err
}

// DefaultBlockBufferSize is the buffer size [WithSizedBlocks] uses when it is
// given a non-positive size.
const DefaultBlockBufferSize = 64 << 10

type arrayWriterOptions struct {
	sized      bool
	bufferSize int
}

// ArrayWriterOption configures an [ArrayWriter].
type ArrayWriterOption func(*arrayWriterOptions)

// WithSizedBlocks makes an [ArrayWriter] emit sized blocks: each block is
// prefixed with its negated item count and its encoded size in bytes, which
// lets a reader discard the whole block with [ArrayReader.SkipBlock] instead of
// decoding it.
//
// A block's size is only known once the block has been encoded, so this buffers
// items. The buffer is flushed as soon as it reaches bufferSize, and therefore
// holds at most bufferSize plus the encoding of a single item. A non-positive
// bufferSize selects [DefaultBlockBufferSize].
func WithSizedBlocks(bufferSize int) ArrayWriterOption {
	return func(o *arrayWriterOptions) {
		o.sized = true
		o.bufferSize = bufferSize
	}
}

// ArrayWriter encodes an Avro array one item at a time.
//
// By default it emits unsized blocks and buffers nothing: each item is written
// straight through as its own block, costing one extra byte per item. Pass
// [WithSizedBlocks] to batch items into sized blocks instead, trading a bounded
// buffer for fewer count prefixes and a stream a reader can skip through.
//
// An array is terminated by a zero-count block, so an ArrayWriter that is never
// closed produces a truncated array rather than a complete one missing a flush.
// [ArrayWriter.Close] must be called, and its error checked; reading such a
// stream back fails with [ErrTruncatedArray]. Use [WriteArray] to have the close
// handled for you.
type ArrayWriter struct {
	w *BinaryWriter

	buf   *bytes.Buffer // pending block; nil when blocks are unsized
	bufw  *BinaryWriter // writes into buf
	limit int           // buffered byte count at which the block is flushed

	count  int64 // items buffered in the pending block
	closed bool
	err    error // sticky error
}

// NewArrayWriter returns an ArrayWriter that encodes an array to w.
func NewArrayWriter(w *BinaryWriter, opts ...ArrayWriterOption) *ArrayWriter {
	o := arrayWriterOptions{bufferSize: DefaultBlockBufferSize}
	for _, opt := range opts {
		opt(&o)
	}

	a := &ArrayWriter{w: w}
	if o.sized {
		if o.bufferSize <= 0 {
			o.bufferSize = DefaultBlockBufferSize
		}
		a.limit = o.bufferSize
		a.buf = bytes.NewBuffer(make([]byte, 0, o.bufferSize))
		a.bufw = NewBinaryWriter(a.buf)
	}
	return a
}

// Write encodes v as the next item of the array. With sized blocks the item is
// buffered and only reaches the underlying writer when the block is flushed.
//
// Once Write returns an error, every later call to Write or
// [ArrayWriter.Close] returns that same error.
func (a *ArrayWriter) Write(v BinaryMarshaler) error {
	if a.err != nil {
		return a.err
	}
	if a.closed {
		return ErrArrayWriterClosed
	}

	if a.buf == nil {
		if err := a.w.WriteLong(1); err != nil {
			return a.fail(err)
		}
		if err := v.MarshalAvroBinary(a.w); err != nil {
			return a.fail(err)
		}
		return nil
	}

	if err := v.MarshalAvroBinary(a.bufw); err != nil {
		return a.fail(err)
	}
	a.count++
	if a.buf.Len() >= a.limit {
		return a.flush()
	}
	return nil
}

// Close flushes any pending block and writes the array's terminating
// zero-count block. Closing an already closed ArrayWriter is a no-op.
func (a *ArrayWriter) Close() error {
	if a.err != nil {
		return a.err
	}
	if a.closed {
		return nil
	}
	if a.buf != nil {
		if err := a.flush(); err != nil {
			return err
		}
	}
	if err := a.w.WriteLong(0); err != nil {
		return a.fail(err)
	}
	a.closed = true
	return nil
}

// flush writes the pending block, if any, as a sized block.
func (a *ArrayWriter) flush() error {
	if a.count == 0 {
		return nil
	}
	if err := a.w.WriteLong(-a.count); err != nil {
		return a.fail(err)
	}
	if err := a.w.WriteLong(int64(a.buf.Len())); err != nil {
		return a.fail(err)
	}
	if err := a.w.WriteFixed(a.buf.Bytes()); err != nil {
		return a.fail(err)
	}
	a.count = 0
	a.buf.Reset()
	return nil
}

func (a *ArrayWriter) fail(err error) error {
	a.err = err
	return err
}

// WriteArray encodes an Avro array to w, calling f with the [ArrayWriter] to
// write items to. It closes the writer, terminating the array, once f returns
// without error.
//
// If f returns an error the array is left unterminated, since a partial array
// should not be presented as a complete one.
func WriteArray(w *BinaryWriter, f func(*ArrayWriter) error, opts ...ArrayWriterOption) error {
	a := NewArrayWriter(w, opts...)
	if err := f(a); err != nil {
		return err
	}
	return a.Close()
}
