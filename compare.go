// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"cmp"
)

// CompareLong compares the zigzag varint encoded longs at the front of a and
// b by the Avro specification's sort order, which orders longs by value.
//
// c is -1 if a's value is less than b's, 0 if they are equal and +1 if a's is
// greater. Overlong encodings compare by value, so 0x80 0x00 equals 0x00.
//
// na and nb are the counts [ConsumeLong] returns for a and b. Each side is
// consumed independently, so a side that fails reports its own negative code
// whatever the other side holds; see [ConsumeError]. c is 0 whenever na or nb
// is negative.
//
// Avro ints share the long encoding, so CompareLong orders encoded ints too.
//
// CompareLong does not allocate.
func CompareLong(a, b []byte) (c, na, nb int) {
	va, na := ConsumeLong(a)
	vb, nb := ConsumeLong(b)
	if na < 0 || nb < 0 {
		return 0, na, nb
	}
	return cmp.Compare(va, vb), na, nb
}

// CompareBytes compares the length prefixed bytes values at the front of a and
// b by the Avro specification's sort order, which orders bytes
// lexicographically as unsigned bytes, as [bytes.Compare] does. A value that
// is a prefix of another sorts first. Strings sort the same way, so
// CompareBytes compares encoded strings too.
//
// c is -1 if a's payload sorts before b's, 0 if they are equal and +1 if a's
// sorts after. The payloads are compared where they lie in a and b, without
// being copied. An overlong length prefix is accepted and decoded to the
// length it encodes, so it does not change the result.
//
// na and nb are the counts [ConsumeBytes] returns for a and b: the length
// prefix plus the payload. Each side is consumed independently, so a side that
// fails reports its own negative code whatever the other side holds; see
// [ConsumeError]. c is 0 whenever na or nb is negative.
//
// CompareBytes does not allocate.
func CompareBytes(a, b []byte) (c, na, nb int) {
	pa, na := ConsumeBytes(a)
	pb, nb := ConsumeBytes(b)
	if na < 0 || nb < 0 {
		return 0, na, nb
	}
	return bytes.Compare(pa, pb), na, nb
}
