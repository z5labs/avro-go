// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// BinaryMarshaler is the interface implemented by types that can marshal themselves into an Avro binary representation.
type BinaryMarshaler interface {
	MarshalAvroBinary(w *BinaryWriter) error
}

// MarshalBinary writes the Avro data to the given writer.
func MarshalBinary(w io.Writer, v BinaryMarshaler) error {
	return v.MarshalAvroBinary(&BinaryWriter{out: w})
}

// BinaryWriter provides methods to write Avro data types to an underlying io.Writer.
type BinaryWriter struct {
	out    io.Writer
	offset int64

	// scratch backs the fixed-size encodings so that they do not escape to the
	// heap on every write. Its contents never outlive a single method call.
	scratch [binary.MaxVarintLen64]byte
}

// NewBinaryWriter returns a BinaryWriter that writes to w.
func NewBinaryWriter(w io.Writer) *BinaryWriter {
	return &BinaryWriter{out: w}
}

// Offset returns the number of bytes successfully written so far.
func (w *BinaryWriter) Offset() int64 {
	return w.offset
}

// BinaryWriterError is returned when a BinaryWriter operation fails. Offset is
// the writer's byte offset before the failed write attempt, BytesWritten is
// the number of bytes that were written during the failed attempt, and Err is
// the underlying cause so callers can use errors.Is and errors.As to inspect it.
type BinaryWriterError struct {
	Offset       int64
	BytesWritten int64
	Err          error
}

func (e *BinaryWriterError) Error() string {
	return fmt.Sprintf("avro: binary writer: offset %d: wrote %d bytes: %v", e.Offset, e.BytesWritten, e.Err)
}

func (e *BinaryWriterError) Unwrap() error {
	return e.Err
}

func (w *BinaryWriter) wrapErr(err error, bytesWritten int) error {
	return &BinaryWriterError{Offset: w.offset, BytesWritten: int64(bytesWritten), Err: err}
}

// WriteBool writes a boolean value to the writer. It writes 1 for true and 0 for false.
func (w *BinaryWriter) WriteBool(b bool) error {
	w.scratch[0] = 0
	if b {
		w.scratch[0] = 1
	}
	n, err := w.out.Write(w.scratch[:1])
	if err != nil {
		return w.wrapErr(err, n)
	}
	if n != 1 {
		return w.wrapErr(io.ErrShortWrite, n)
	}
	w.offset += int64(n)
	return nil
}

// WriteInt writes a 32-bit integer to the writer using variable-length zigzag encoding.
func (w *BinaryWriter) WriteInt(i int32) error {
	return w.WriteLong(int64(i))
}

// WriteLong writes a 64-bit integer to the writer using variable-length zigzag encoding.
func (w *BinaryWriter) WriteLong(l int64) error {
	n := binary.PutVarint(w.scratch[:], l)
	nw, err := w.out.Write(w.scratch[:n])
	if err != nil {
		return w.wrapErr(err, nw)
	}
	if nw != n {
		return w.wrapErr(io.ErrShortWrite, nw)
	}
	w.offset += int64(nw)
	return nil
}

// WriteFloat writes a 32-bit floating-point number to the writer in little-endian format.
func (w *BinaryWriter) WriteFloat(f float32) error {
	binary.LittleEndian.PutUint32(w.scratch[:4], math.Float32bits(f))
	nw, err := w.out.Write(w.scratch[:4])
	if err != nil {
		return w.wrapErr(err, nw)
	}
	if nw != 4 {
		return w.wrapErr(io.ErrShortWrite, nw)
	}
	w.offset += int64(nw)
	return nil
}

// WriteDouble writes a 64-bit floating-point number to the writer in little-endian format.
func (w *BinaryWriter) WriteDouble(d float64) error {
	binary.LittleEndian.PutUint64(w.scratch[:8], math.Float64bits(d))
	nw, err := w.out.Write(w.scratch[:8])
	if err != nil {
		return w.wrapErr(err, nw)
	}
	if nw != 8 {
		return w.wrapErr(io.ErrShortWrite, nw)
	}
	w.offset += int64(nw)
	return nil
}

// WriteBytes writes a byte array to the writer. It first writes the length of the array as a long, followed by the bytes.
func (w *BinaryWriter) WriteBytes(b []byte) error {
	if err := w.WriteLong(int64(len(b))); err != nil {
		return err
	}
	nw, err := w.out.Write(b)
	if err != nil {
		return w.wrapErr(err, nw)
	}
	if nw != len(b) {
		return w.wrapErr(io.ErrShortWrite, nw)
	}
	w.offset += int64(nw)
	return nil
}

// WriteFixed writes a fixed number of bytes to the writer. Unlike WriteBytes, it does not write a length prefix.
func (w *BinaryWriter) WriteFixed(b []byte) error {
	nw, err := w.out.Write(b)
	if err != nil {
		return w.wrapErr(err, nw)
	}
	if nw != len(b) {
		return w.wrapErr(io.ErrShortWrite, nw)
	}
	w.offset += int64(nw)
	return nil
}

// WriteString writes a string to the writer. It first writes the length of the string as a long, followed by the UTF-8 bytes of the string.
func (w *BinaryWriter) WriteString(s string) error {
	if err := w.WriteLong(int64(len(s))); err != nil {
		return err
	}
	nw, err := io.WriteString(w.out, s)
	if err != nil {
		return w.wrapErr(err, nw)
	}
	if nw != len(s) {
		return w.wrapErr(io.ErrShortWrite, nw)
	}
	w.offset += int64(nw)
	return nil
}

// BinaryUnmarshaler is the interface implemented by types that can unmarshal an Avro binary representation of themselves.
type BinaryUnmarshaler interface {
	UnmarshalAvroBinary(r *BinaryReader) error
}

// UnmarshalBinary reads Avro data from the given reader.
func UnmarshalBinary(r io.Reader, v BinaryUnmarshaler) error {
	return v.UnmarshalAvroBinary(&BinaryReader{in: r})
}

// UnmarshalBytes reads Avro data from b, using a [BinaryReader] in bytes mode;
// see [NewBytesReader]. Bytes left over after v has been read are ignored, as
// [UnmarshalBinary] ignores the rest of its reader.
//
// UnmarshalBytes allocates the BinaryReader it hands to v on every call. To
// decode without allocating, create one reader with [NewBytesReader] and call
// [BinaryReader.ResetBytes] on it before each record instead.
//
// The []byte values v reads alias b, as they do for [NewBytesReader].
func UnmarshalBytes(b []byte, v BinaryUnmarshaler) error {
	return v.UnmarshalAvroBinary(NewBytesReader(b))
}

// BinaryReader provides methods to read Avro data types from an underlying
// io.Reader, or straight from a []byte.
//
// A BinaryReader made by [NewBinaryReader] reads from an io.Reader. One made
// by [NewBytesReader], or reset with [BinaryReader.ResetBytes], is in bytes
// mode: it decodes from a []byte with the Consume functions, such as
// [ConsumeLong], and never touches an io.Reader. In bytes mode:
//
//   - [BinaryReader.ReadBytes] and [BinaryReader.ReadFixed] return slices of
//     the input rather than copies. Their capacity equals their length, so
//     appending to one copies instead of overwriting the input, but they are
//     valid only as long as the input is. [BinaryReader.ReadString] still
//     copies.
//   - A read that fails consumes nothing: [BinaryReader.Offset] and
//     [BinaryReader.Remaining] stay at the start of the value that failed, and
//     so does the Offset of the [*BinaryReaderError] it returns.
//   - A read returns [io.EOF] if no input remains when it starts, and
//     [io.ErrUnexpectedEOF] if the input ends partway through the value.
//   - Every read except [BinaryReader.ReadString] makes no allocation on
//     success, so a record decoded through one reused reader allocates
//     nothing but its strings.
type BinaryReader struct {
	in     io.Reader
	offset int64

	// isBytes reports bytes mode, in which rest holds the input not yet
	// consumed and in is unused.
	isBytes bool
	rest    []byte

	// scratch backs the fixed-size decodings so that they do not escape to the
	// heap on every read. Its contents never outlive a single method call, and
	// are never handed to a caller.
	scratch [8]byte
}

// NewBinaryReader returns a BinaryReader that reads from r.
func NewBinaryReader(r io.Reader) *BinaryReader {
	return &BinaryReader{in: r}
}

// NewBytesReader returns a BinaryReader in bytes mode that reads from b; see
// [BinaryReader] for how bytes mode differs. The reader does not copy b, and
// the []byte values it returns alias it.
func NewBytesReader(b []byte) *BinaryReader {
	return &BinaryReader{isBytes: true, rest: b}
}

// ResetBytes puts r in bytes mode reading from b, discarding whatever r was
// reading before, and resets [BinaryReader.Offset] to zero. Reusing one
// reader across records with ResetBytes is what lets a record decode with no
// allocations but its strings.
func (r *BinaryReader) ResetBytes(b []byte) {
	*r = BinaryReader{isBytes: true, rest: b}
}

// Offset returns the number of bytes successfully consumed so far. In bytes
// mode it counts from the last call to [NewBytesReader] or
// [BinaryReader.ResetBytes], so that Offset equals the input's length minus
// the length of [BinaryReader.Remaining].
func (r *BinaryReader) Offset() int64 {
	return r.offset
}

// Remaining returns the input a bytes-mode reader has not yet consumed. The
// result aliases the input. It returns nil when r reads from an io.Reader.
func (r *BinaryReader) Remaining() []byte {
	if !r.isBytes {
		return nil
	}
	return r.rest
}

// consumed advances a bytes-mode reader past the n bytes a Consume function
// reported, or returns the error it reported. On error nothing is consumed.
func (r *BinaryReader) consumed(n int) error {
	if n < 0 {
		if len(r.rest) == 0 {
			return r.wrapErr(io.EOF)
		}
		return r.wrapErr(ConsumeError(n))
	}
	r.rest = r.rest[n:]
	r.offset += int64(n)
	return nil
}

// BinaryReaderError is returned when a BinaryReader operation fails. It reports
// the byte offset at which the error occurred and wraps the underlying cause
// so callers can use errors.Is and errors.As to inspect it.
type BinaryReaderError struct {
	Offset int64
	Err    error
}

func (e *BinaryReaderError) Error() string {
	return fmt.Sprintf("avro: binary reader: offset %d: %v", e.Offset, e.Err)
}

func (e *BinaryReaderError) Unwrap() error {
	return e.Err
}

func (r *BinaryReader) wrapErr(err error) error {
	return &BinaryReaderError{Offset: r.offset, Err: err}
}

var (
	// ErrNegativeLength is returned when a decoded length value is negative.
	ErrNegativeLength = errors.New("avro: negative length")

	// ErrOverflow is returned when a varint or integer value exceeds the expected range.
	ErrOverflow = errors.New("avro: varint overflow")
)

// ReadBool reads a boolean value from the reader. It returns true if the byte is non-zero.
func (r *BinaryReader) ReadBool() (bool, error) {
	if r.isBytes {
		v, n := ConsumeBool(r.rest)
		return v, r.consumed(n)
	}
	n, err := io.ReadFull(r.in, r.scratch[:1])
	r.offset += int64(n)
	if err != nil {
		return false, r.wrapErr(err)
	}
	return r.scratch[0] != 0, nil
}

// ReadInt reads a 32-bit integer from the reader using variable-length zigzag encoding.
//
// It follows the same overflow and end-of-input rules as [BinaryReader.ReadLong],
// and also returns [ErrOverflow] when the decoded value does not fit in 32 bits.
func (r *BinaryReader) ReadInt() (int32, error) {
	if r.isBytes {
		v, n := ConsumeInt(r.rest)
		return v, r.consumed(n)
	}
	l, err := r.readLong()
	if err != nil {
		return 0, err
	}
	if l < math.MinInt32 || l > math.MaxInt32 {
		return 0, r.wrapErr(ErrOverflow)
	}
	return int32(l), nil
}

// ReadLong reads a 64-bit integer from the reader using variable-length zigzag encoding.
//
// It returns [ErrOverflow] when the varint encodes a value that does not fit
// in 64 bits. It returns [io.EOF] only when no byte of the value could be
// read, and [io.ErrUnexpectedEOF] when the input ends partway through it.
func (r *BinaryReader) ReadLong() (int64, error) {
	if r.isBytes {
		v, n := ConsumeLong(r.rest)
		return v, r.consumed(n)
	}
	return r.readLong()
}

// readLong is [BinaryReader.ReadLong] from an io.Reader. The other methods
// that begin with a long call it directly, so that they check the mode once.
func (r *BinaryReader) readLong() (int64, error) {
	var unsigned uint64
	var shift uint
	for i := 0; i < binary.MaxVarintLen64; i++ {
		n, err := io.ReadFull(r.in, r.scratch[:1])
		r.offset += int64(n)
		if err != nil {
			if i > 0 && errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return 0, r.wrapErr(err)
		}
		b := r.scratch[0]
		// The 10th byte carries bit 63 alone; anything above 0x01 encodes a
		// value of at least 2^64, or continues past the maximum varint length.
		if i == binary.MaxVarintLen64-1 && b > 0x01 {
			return 0, r.wrapErr(ErrOverflow)
		}
		unsigned |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return int64(unsigned>>1) ^ -int64(unsigned&1), nil
		}
		shift += 7
	}
	// Unreachable: the 10th byte either overflows above or ends the varint.
	return 0, r.wrapErr(ErrOverflow)
}

// ReadIndex reads an enum symbol or union branch index from the reader, where
// count is the number of symbols or branches. The index is encoded as a long.
//
// It follows the same overflow and end-of-input rules as [BinaryReader.ReadLong].
// When the index is negative or not less than count, it returns a
// [*BinaryReaderError] wrapping [ErrInvalidIndex], whose Offset is just past
// the index; in bytes mode the index is not consumed, and the Offset is its
// start. It is the [BinaryReader] counterpart of [ConsumeIndex].
func (r *BinaryReader) ReadIndex(count int) (int, error) {
	if r.isBytes {
		v, n := ConsumeIndex(r.rest, count)
		return v, r.consumed(n)
	}
	v, err := r.readLong()
	if err != nil {
		return 0, err
	}
	if v < 0 || v >= int64(count) {
		return 0, r.wrapErr(ErrInvalidIndex)
	}
	return int(v), nil
}

// ReadFloat reads a 32-bit floating-point number from the reader in little-endian format.
func (r *BinaryReader) ReadFloat() (float32, error) {
	if r.isBytes {
		v, n := ConsumeFloat(r.rest)
		return v, r.consumed(n)
	}
	n, err := io.ReadFull(r.in, r.scratch[:4])
	r.offset += int64(n)
	if err != nil {
		return 0, r.wrapErr(err)
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(r.scratch[:4])), nil
}

// ReadDouble reads a 64-bit floating-point number from the reader in little-endian format.
func (r *BinaryReader) ReadDouble() (float64, error) {
	if r.isBytes {
		v, n := ConsumeDouble(r.rest)
		return v, r.consumed(n)
	}
	n, err := io.ReadFull(r.in, r.scratch[:8])
	r.offset += int64(n)
	if err != nil {
		return 0, r.wrapErr(err)
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(r.scratch[:8])), nil
}

// ReadBytes reads a byte array from the reader. It first reads the length as a long, followed by the bytes.
//
// It returns [io.EOF] only when no byte of the length prefix could be read,
// and [io.ErrUnexpectedEOF] when the input ends after that.
//
// In bytes mode the result is a slice of the input, not a copy; see
// [BinaryReader]. An empty value is a non-nil, zero-length slice in either
// mode.
func (r *BinaryReader) ReadBytes() ([]byte, error) {
	if r.isBytes {
		v, n := ConsumeBytes(r.rest)
		return v, r.consumed(n)
	}
	n, err := r.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, r.wrapErr(ErrNegativeLength)
	}
	if n > int64(math.MaxInt32) {
		return nil, r.wrapErr(ErrOverflow)
	}
	if n == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, n)
	nr, err := io.ReadFull(r.in, buf)
	r.offset += int64(nr)
	if err != nil {
		// The length prefix has already been consumed, so the value is cut
		// short even when no byte of its payload was available.
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, r.wrapErr(err)
	}
	return buf, nil
}

// ReadFixed reads exactly size bytes from the reader. Unlike ReadBytes, it does not read a length prefix.
//
// In bytes mode the result is a slice of the input, not a copy; see
// [BinaryReader]. A size of 0 succeeds without consuming anything in either
// mode, even when no input remains. ReadFixed panics if size is negative.
func (r *BinaryReader) ReadFixed(size int) ([]byte, error) {
	if r.isBytes {
		v, n := ConsumeFixed(r.rest, size)
		if v == nil && n == 0 {
			// A size of 0 from nil input. Return a non-nil empty slice, as
			// an io.Reader-mode read does.
			v = []byte{}
		}
		return v, r.consumed(n)
	}
	buf := make([]byte, size)
	n, err := io.ReadFull(r.in, buf)
	r.offset += int64(n)
	if err != nil {
		return nil, r.wrapErr(err)
	}
	return buf, nil
}

// discard advances the reader past n bytes without retaining them.
func (r *BinaryReader) discard(n int64) error {
	if r.isBytes {
		if n < 0 {
			// io.CopyN discards nothing for a negative n; do the same.
			return nil
		}
		if n > int64(len(r.rest)) {
			// The bytes being discarded belong to a value that has already
			// begun, so running out of them is a short read.
			return r.wrapErr(io.ErrUnexpectedEOF)
		}
		_, c := ConsumeFixed(r.rest, int(n))
		return r.consumed(c)
	}
	nr, err := io.CopyN(io.Discard, r.in, n)
	r.offset += nr
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return r.wrapErr(err)
	}
	return nil
}

// ReadString reads a string from the reader. It first reads the length as a long, followed by the UTF-8 bytes of the string.
//
// It follows the same end-of-input rules as [BinaryReader.ReadBytes]. It
// copies the bytes into a new string in bytes mode too.
func (r *BinaryReader) ReadString() (string, error) {
	b, err := r.ReadBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}
