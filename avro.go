// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"encoding/binary"
	"io"
	"math"
)

type BinaryMarshaler interface {
	MarshalAvroBinary(w *BinaryWriter) error
}

// MarshalBinary writes the Avro data to the given writer.
func MarshalBinary(w io.Writer, v BinaryMarshaler) error {
	return v.MarshalAvroBinary(&BinaryWriter{out: w})
}

// BinaryWriter provides methods to write Avro data types to an underlying io.Writer.
type BinaryWriter struct {
	out io.Writer
}

// WriteBool writes a boolean value to the writer. It writes 1 for true and 0 for false.
func (w *BinaryWriter) WriteBool(b bool) error {
	var value byte
	if b {
		value = 1
	}
	n, err := w.out.Write([]byte{value})
	if err != nil {
		return err
	}
	if n != 1 {
		return io.ErrShortWrite
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
	if err != nil {
		return err
	}
	if nw != n {
		return io.ErrShortWrite
	}
	return nil
}

// WriteFloat writes a 32-bit floating-point number to the writer in little-endian format.
func (w *BinaryWriter) WriteFloat(f float32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(f))
	nw, err := w.out.Write(buf[:])
	if err != nil {
		return err
	}
	if nw != 4 {
		return io.ErrShortWrite
	}
	return nil
}

// WriteDouble writes a 64-bit floating-point number to the writer in little-endian format.
func (w *BinaryWriter) WriteDouble(d float64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(d))
	nw, err := w.out.Write(buf[:])
	if err != nil {
		return err
	}
	if nw != 8 {
		return io.ErrShortWrite
	}
	return nil
}

// WriteBytes writes a byte array to the writer. It first writes the length of the array as a long, followed by the bytes.
func (w *BinaryWriter) WriteBytes(b []byte) error {
	err := w.WriteLong(int64(len(b)))
	if err != nil {
		return err
	}
	nw, err := w.out.Write(b)
	if err != nil {
		return err
	}
	if nw != len(b) {
		return io.ErrShortWrite
	}
	return nil
}

// WriteString writes a string to the writer. It first writes the length of the string as a long, followed by the UTF-8 bytes of the string.
func (w *BinaryWriter) WriteString(s string) error {
	err := w.WriteLong(int64(len(s)))
	if err != nil {
		return err
	}
	nw, err := io.WriteString(w.out, s)
	if err != nil {
		return err
	}
	if nw != len(s) {
		return io.ErrShortWrite
	}
	return nil
}
