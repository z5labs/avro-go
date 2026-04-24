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
}

// Offset returns the number of bytes successfully written so far.
func (w *BinaryWriter) Offset() int64 {
	return w.offset
}

// BinaryWriterError is returned when a BinaryWriter operation fails. It reports
// the byte offset at which the error occurred and wraps the underlying cause
// so callers can use errors.Is and errors.As to inspect it.
type BinaryWriterError struct {
	Offset int64
	Err    error
}

func (e *BinaryWriterError) Error() string {
	return fmt.Sprintf("avro: binary writer: offset %d: %v", e.Offset, e.Err)
}

func (e *BinaryWriterError) Unwrap() error {
	return e.Err
}

func (w *BinaryWriter) wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return &BinaryWriterError{Offset: w.offset, Err: err}
}

// WriteBool writes a boolean value to the writer. It writes 1 for true and 0 for false.
func (w *BinaryWriter) WriteBool(b bool) error {
	var value byte
	if b {
		value = 1
	}
	n, err := w.out.Write([]byte{value})
	w.offset += int64(n)
	if err != nil {
		return w.wrapErr(err)
	}
	if n != 1 {
		return w.wrapErr(io.ErrShortWrite)
	}
	return nil
}

// WriteInt writes a 32-bit integer to the writer using variable-length zigzag encoding.
func (w *BinaryWriter) WriteInt(i int32) error {
	return w.WriteLong(int64(i))
}

// WriteLong writes a 64-bit integer to the writer using variable-length zigzag encoding.
func (w *BinaryWriter) WriteLong(l int64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutVarint(buf[:], l)
	nw, err := w.out.Write(buf[:n])
	w.offset += int64(nw)
	if err != nil {
		return w.wrapErr(err)
	}
	if nw != n {
		return w.wrapErr(io.ErrShortWrite)
	}
	return nil
}

// WriteFloat writes a 32-bit floating-point number to the writer in little-endian format.
func (w *BinaryWriter) WriteFloat(f float32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(f))
	nw, err := w.out.Write(buf[:])
	w.offset += int64(nw)
	if err != nil {
		return w.wrapErr(err)
	}
	if nw != 4 {
		return w.wrapErr(io.ErrShortWrite)
	}
	return nil
}

// WriteDouble writes a 64-bit floating-point number to the writer in little-endian format.
func (w *BinaryWriter) WriteDouble(d float64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(d))
	nw, err := w.out.Write(buf[:])
	w.offset += int64(nw)
	if err != nil {
		return w.wrapErr(err)
	}
	if nw != 8 {
		return w.wrapErr(io.ErrShortWrite)
	}
	return nil
}

// WriteBytes writes a byte array to the writer. It first writes the length of the array as a long, followed by the bytes.
func (w *BinaryWriter) WriteBytes(b []byte) error {
	if err := w.WriteLong(int64(len(b))); err != nil {
		return err
	}
	nw, err := w.out.Write(b)
	w.offset += int64(nw)
	if err != nil {
		return w.wrapErr(err)
	}
	if nw != len(b) {
		return w.wrapErr(io.ErrShortWrite)
	}
	return nil
}

// WriteFixed writes a fixed number of bytes to the writer. Unlike WriteBytes, it does not write a length prefix.
func (w *BinaryWriter) WriteFixed(b []byte) error {
	nw, err := w.out.Write(b)
	w.offset += int64(nw)
	if err != nil {
		return w.wrapErr(err)
	}
	if nw != len(b) {
		return w.wrapErr(io.ErrShortWrite)
	}
	return nil
}

// WriteString writes a string to the writer. It first writes the length of the string as a long, followed by the UTF-8 bytes of the string.
func (w *BinaryWriter) WriteString(s string) error {
	if err := w.WriteLong(int64(len(s))); err != nil {
		return err
	}
	nw, err := io.WriteString(w.out, s)
	w.offset += int64(nw)
	if err != nil {
		return w.wrapErr(err)
	}
	if nw != len(s) {
		return w.wrapErr(io.ErrShortWrite)
	}
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

// BinaryReader provides methods to read Avro data types from an underlying io.Reader.
type BinaryReader struct {
	in     io.Reader
	offset int64
}

// Offset returns the number of bytes successfully consumed so far.
func (r *BinaryReader) Offset() int64 {
	return r.offset
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
	if err == nil {
		return nil
	}
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
	var buf [1]byte
	n, err := io.ReadFull(r.in, buf[:])
	r.offset += int64(n)
	if err != nil {
		return false, r.wrapErr(err)
	}
	return buf[0] != 0, nil
}

// ReadInt reads a 32-bit integer from the reader using variable-length zigzag encoding.
func (r *BinaryReader) ReadInt() (int32, error) {
	l, err := r.ReadLong()
	if err != nil {
		return 0, err
	}
	if l < math.MinInt32 || l > math.MaxInt32 {
		return 0, r.wrapErr(ErrOverflow)
	}
	return int32(l), nil
}

// ReadLong reads a 64-bit integer from the reader using variable-length zigzag encoding.
func (r *BinaryReader) ReadLong() (int64, error) {
	var buf [1]byte
	var unsigned uint64
	var shift uint
	for i := 0; i < binary.MaxVarintLen64; i++ {
		n, err := io.ReadFull(r.in, buf[:])
		r.offset += int64(n)
		if err != nil {
			return 0, r.wrapErr(err)
		}
		b := buf[0]
		unsigned |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return int64(unsigned>>1) ^ -int64(unsigned&1), nil
		}
		shift += 7
	}
	return 0, r.wrapErr(ErrOverflow)
}

// ReadFloat reads a 32-bit floating-point number from the reader in little-endian format.
func (r *BinaryReader) ReadFloat() (float32, error) {
	var buf [4]byte
	n, err := io.ReadFull(r.in, buf[:])
	r.offset += int64(n)
	if err != nil {
		return 0, r.wrapErr(err)
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(buf[:])), nil
}

// ReadDouble reads a 64-bit floating-point number from the reader in little-endian format.
func (r *BinaryReader) ReadDouble() (float64, error) {
	var buf [8]byte
	n, err := io.ReadFull(r.in, buf[:])
	r.offset += int64(n)
	if err != nil {
		return 0, r.wrapErr(err)
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(buf[:])), nil
}

// ReadBytes reads a byte array from the reader. It first reads the length as a long, followed by the bytes.
func (r *BinaryReader) ReadBytes() ([]byte, error) {
	n, err := r.ReadLong()
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
		return nil, r.wrapErr(err)
	}
	return buf, nil
}

// ReadFixed reads exactly size bytes from the reader. Unlike ReadBytes, it does not read a length prefix.
func (r *BinaryReader) ReadFixed(size int) ([]byte, error) {
	buf := make([]byte, size)
	n, err := io.ReadFull(r.in, buf)
	r.offset += int64(n)
	if err != nil {
		return nil, r.wrapErr(err)
	}
	return buf, nil
}

// ReadString reads a string from the reader. It first reads the length as a long, followed by the UTF-8 bytes of the string.
func (r *BinaryReader) ReadString() (string, error) {
	b, err := r.ReadBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}
