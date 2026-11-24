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
	"math/bits"
)

// This file holds the []byte counterparts of [BinaryReader] and
// [BinaryWriter]: free functions that decode Avro primitives from the front of
// a byte slice and append their encodings to one, with no io.Reader or
// io.Writer, no interface calls and no allocation.
//
// The Consume functions share one result convention. A count n >= 0 is
// success, and is the number of bytes of b the value occupied. A negative n is
// an error code, which only [ConsumeError] interprets. When n < 0 the decoded
// value is the zero value.
//
// The bytes and fixed results of [ConsumeBytes] and [ConsumeFixed] alias b
// rather than copying it. Their capacity equals their length, so appending to
// one copies instead of overwriting the rest of b. They are valid only as long
// as b is: reusing or overwriting b's backing array changes them.

// ErrInvalidIndex is returned when a decoded enum symbol or union branch index
// is negative or not less than the number of symbols or branches.
var ErrInvalidIndex = errors.New("avro: enum or union index out of range")

// Error codes returned as a negative n by the Consume functions.
const (
	errCodeTruncated      = -1
	errCodeOverflow       = -2
	errCodeNegativeLength = -3
	errCodeInvalidIndex   = -4
)

// ConsumeError returns the error a Consume function reported with n.
//
// It returns nil when n >= 0. Otherwise it returns [io.ErrUnexpectedEOF],
// [ErrOverflow], [ErrNegativeLength] or [ErrInvalidIndex]. Input that ends
// before the value does, including empty input, is [io.ErrUnexpectedEOF]. The
// errors are the package's sentinels, so ConsumeError never allocates.
//
// A negative n that no Consume function returns is reported as
// [io.ErrUnexpectedEOF].
func ConsumeError(n int) error {
	switch {
	case n >= 0:
		return nil
	case n == errCodeOverflow:
		return ErrOverflow
	case n == errCodeNegativeLength:
		return ErrNegativeLength
	case n == errCodeInvalidIndex:
		return ErrInvalidIndex
	default:
		return io.ErrUnexpectedEOF
	}
}

// ConsumeLong decodes a zigzag varint encoded long from the front of b.
//
// It reports [ErrOverflow] when the 10th byte is above 0x01, which covers a
// value that does not fit in 64 bits and a varint that continues past 10
// bytes. Overlong encodings, such as 0x80 0x00 for 0, are accepted, as
// [BinaryReader.ReadLong] accepts them. See [ConsumeError] for the meaning of
// n.
func ConsumeLong(b []byte) (v int64, n int) {
	var u uint64
	var shift uint
	for i, c := range b {
		// The 10th byte carries bit 63 alone; anything above 0x01 encodes a
		// value of at least 2^64, or continues past the maximum varint length.
		if i == binary.MaxVarintLen64-1 && c > 0x01 {
			return 0, errCodeOverflow
		}
		u |= uint64(c&0x7f) << shift
		if c < 0x80 {
			return int64(u>>1) ^ -int64(u&1), i + 1
		}
		shift += 7
	}
	return 0, errCodeTruncated
}

// ConsumeInt decodes a zigzag varint encoded int from the front of b.
//
// It follows the same rules as [ConsumeLong], and also reports [ErrOverflow]
// when the value does not fit in 32 bits, as [BinaryReader.ReadInt] does.
func ConsumeInt(b []byte) (v int32, n int) {
	l, n := ConsumeLong(b)
	if n < 0 {
		return 0, n
	}
	if l < math.MinInt32 || l > math.MaxInt32 {
		return 0, errCodeOverflow
	}
	return int32(l), n
}

// ConsumeLength decodes the long length prefix of a bytes or string value
// from the front of b. It consumes the prefix only, not the payload.
//
// It reports [ErrNegativeLength] for a negative length and [ErrOverflow] for
// a length above [math.MaxInt32], as [BinaryReader.ReadBytes] does.
func ConsumeLength(b []byte) (l int, n int) {
	v, n := ConsumeLong(b)
	if n < 0 {
		return 0, n
	}
	if v < 0 {
		return 0, errCodeNegativeLength
	}
	if v > math.MaxInt32 {
		return 0, errCodeOverflow
	}
	return int(v), n
}

// ConsumeIndex decodes an enum symbol or union branch index from the front of
// b, where count is the number of symbols or branches.
//
// It reports [ErrInvalidIndex] when the index is negative or not less than
// count.
func ConsumeIndex(b []byte, count int) (i int, n int) {
	v, n := ConsumeLong(b)
	if n < 0 {
		return 0, n
	}
	if v < 0 || v >= int64(count) {
		return 0, errCodeInvalidIndex
	}
	return int(v), n
}

// ConsumeBytes decodes a length prefixed bytes value from the front of b. It
// decodes a string the same way.
//
// The result aliases b and its capacity equals its length; reusing or
// overwriting b invalidates it. An empty value is a non-nil, zero-length
// slice. The length prefix follows the rules of [ConsumeLength].
func ConsumeBytes(b []byte) (v []byte, n int) {
	l, n := ConsumeLength(b)
	if n < 0 {
		return nil, n
	}
	if l > len(b)-n {
		return nil, errCodeTruncated
	}
	return b[n : n+l : n+l], n + l
}

// ConsumeFixed decodes a fixed value of size bytes from the front of b.
//
// The result aliases b and its capacity equals its length; reusing or
// overwriting b invalidates it. A size of 0 succeeds with n == 0.
// ConsumeFixed panics if size is negative.
func ConsumeFixed(b []byte, size int) (v []byte, n int) {
	if size < 0 {
		panic("avro: ConsumeFixed: negative size")
	}
	if size > len(b) {
		return nil, errCodeTruncated
	}
	return b[:size:size], size
}

// ConsumeBool decodes a boolean from the front of b. Any non-zero byte is
// true, as it is for [BinaryReader.ReadBool].
func ConsumeBool(b []byte) (v bool, n int) {
	if len(b) < 1 {
		return false, errCodeTruncated
	}
	return b[0] != 0, 1
}

// ConsumeFloat decodes a little-endian 32-bit float from the front of b.
func ConsumeFloat(b []byte) (v float32, n int) {
	if len(b) < 4 {
		return 0, errCodeTruncated
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), 4
}

// ConsumeDouble decodes a little-endian 64-bit float from the front of b.
func ConsumeDouble(b []byte) (v float64, n int) {
	if len(b) < 8 {
		return 0, errCodeTruncated
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), 8
}

// zigzag maps a signed long onto the unsigned value Avro varint encodes.
func zigzag(v int64) uint64 {
	return uint64(v<<1) ^ uint64(v>>63)
}

// AppendLong appends the zigzag varint encoding of v to dst, the bytes
// [BinaryWriter.WriteLong] writes.
func AppendLong(dst []byte, v int64) []byte {
	return binary.AppendUvarint(dst, zigzag(v))
}

// AppendInt appends the zigzag varint encoding of v to dst, the bytes
// [BinaryWriter.WriteInt] writes.
func AppendInt(dst []byte, v int32) []byte {
	return AppendLong(dst, int64(v))
}

// AppendIndex appends the encoding of an enum symbol or union branch index to
// dst. An index is encoded as a long.
func AppendIndex(dst []byte, i int) []byte {
	return AppendLong(dst, int64(i))
}

// AppendBool appends the encoding of v to dst: 1 for true and 0 for false, as
// [BinaryWriter.WriteBool] writes.
func AppendBool(dst []byte, v bool) []byte {
	if v {
		return append(dst, 1)
	}
	return append(dst, 0)
}

// AppendFloat appends the little-endian encoding of v to dst, the bytes
// [BinaryWriter.WriteFloat] writes.
func AppendFloat(dst []byte, v float32) []byte {
	return binary.LittleEndian.AppendUint32(dst, math.Float32bits(v))
}

// AppendDouble appends the little-endian encoding of v to dst, the bytes
// [BinaryWriter.WriteDouble] writes.
func AppendDouble(dst []byte, v float64) []byte {
	return binary.LittleEndian.AppendUint64(dst, math.Float64bits(v))
}

// AppendBytes appends the length prefixed encoding of v to dst, the bytes
// [BinaryWriter.WriteBytes] writes.
func AppendBytes(dst, v []byte) []byte {
	return append(AppendLong(dst, int64(len(v))), v...)
}

// AppendString appends the length prefixed encoding of v to dst, the bytes
// [BinaryWriter.WriteString] writes.
func AppendString(dst []byte, v string) []byte {
	return append(AppendLong(dst, int64(len(v))), v...)
}

// AppendFixed appends v to dst with no length prefix, the bytes
// [BinaryWriter.WriteFixed] writes.
func AppendFixed(dst, v []byte) []byte {
	return append(dst, v...)
}

// SizeLong returns the number of bytes [AppendLong] appends for v.
func SizeLong(v int64) int {
	// The division-free form of ceil(bits/7), with 0 taking one byte, from
	// google.golang.org/protobuf/encoding/protowire.SizeVarint.
	return int(9*uint32(bits.Len64(zigzag(v)))+64) / 64
}

// SizeIndex returns the number of bytes [AppendIndex] appends for i.
func SizeIndex(i int) int {
	return SizeLong(int64(i))
}

// SizeBytes returns the number of bytes [AppendBytes] or [AppendString]
// appends for a value of n bytes: its length prefix plus n.
func SizeBytes(n int) int {
	return SizeLong(int64(n)) + n
}
