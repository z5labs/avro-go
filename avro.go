// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"encoding/binary"
	"errors"
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
	in io.Reader
}

var errNegativeLength = errors.New("avro: negative length")

// ReadBool reads a boolean value from the reader. It returns true if the byte is non-zero.
func (r *BinaryReader) ReadBool() (bool, error) {
	var buf [1]byte
	_, err := io.ReadFull(r.in, buf[:])
	if err != nil {
		return false, err
	}
	return buf[0] != 0, nil
}

// ReadInt reads a 32-bit integer from the reader using variable-length zigzag encoding.
func (r *BinaryReader) ReadInt() (int32, error) {
	l, err := r.ReadLong()
	if err != nil {
		return 0, err
	}
	return int32(l), nil
}

// ReadLong reads a 64-bit integer from the reader using variable-length zigzag encoding.
func (r *BinaryReader) ReadLong() (int64, error) {
	var buf [1]byte
	var unsigned uint64
	var shift uint
	for {
		_, err := io.ReadFull(r.in, buf[:])
		if err != nil {
			return 0, err
		}
		b := buf[0]
		unsigned |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return int64(unsigned>>1) ^ -int64(unsigned&1), nil
}

// ReadFloat reads a 32-bit floating-point number from the reader in little-endian format.
func (r *BinaryReader) ReadFloat() (float32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r.in, buf[:])
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(buf[:])), nil
}

// ReadDouble reads a 64-bit floating-point number from the reader in little-endian format.
func (r *BinaryReader) ReadDouble() (float64, error) {
	var buf [8]byte
	_, err := io.ReadFull(r.in, buf[:])
	if err != nil {
		return 0, err
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
		return nil, errNegativeLength
	}
	if n == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(r.in, buf)
	if err != nil {
		return nil, err
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
