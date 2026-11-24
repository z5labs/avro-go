// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"cmp"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// decodeSide is the result of decoding one side of a comparison through
// BinaryReader, the path CompareLong and CompareBytes must agree with.
type decodeSide[T any] struct {
	v   T
	n   int
	err error
}

func decodeWith[T any](b []byte, read func(r *BinaryReader) (T, error)) decodeSide[T] {
	r := NewBinaryReader(bytes.NewReader(b))
	v, err := read(r)
	if err != nil {
		// BinaryReader reports empty input as io.EOF, where the Consume
		// convention reports all input that ends before the value does as
		// io.ErrUnexpectedEOF.
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return decodeSide[T]{err: err}
	}
	return decodeSide[T]{v: v, n: int(r.Offset())}
}

// requireSide asserts that n is the count the Consume convention gives for a
// side that decoded as d.
func requireSide[T any](t *testing.T, d decodeSide[T], n int) {
	t.Helper()

	if d.err != nil {
		require.Negative(t, n)
		require.ErrorIs(t, d.err, ConsumeError(n))
		return
	}
	require.Equal(t, d.n, n)
}

// requireCompare asserts that the result of a Compare function agrees with
// decoding both sides first and comparing the values with compare.
func requireCompare[T any](t *testing.T, da, db decodeSide[T], compare func(T, T) int, c, na, nb int) {
	t.Helper()

	requireSide(t, da, na)
	requireSide(t, db, nb)
	if da.err != nil || db.err != nil {
		require.Zero(t, c)
		return
	}
	require.Equal(t, compare(da.v, db.v), c)
}

// overlong returns an overlong encoding of v: its varint padded with extra
// bytes that carry no value bits. extra must be at least 1.
func overlong(v int64, extra int) []byte {
	if extra < 1 {
		panic("overlong: extra must be at least 1")
	}
	b := AppendLong(nil, v)
	b[len(b)-1] |= 0x80
	for range extra - 1 {
		b = append(b, 0x80)
	}
	return append(b, 0x00)
}

func TestCompareLong(t *testing.T) {
	t.Parallel()

	tenthByteOverflow := append(bytes.Repeat([]byte{0xff}, 9), 0x02)

	testCases := []struct {
		name     string
		a        []byte
		b        []byte
		expected int
	}{
		{
			name:     "equal zeros",
			a:        AppendLong(nil, 0),
			b:        AppendLong(nil, 0),
			expected: 0,
		},
		{
			name:     "smaller positive first",
			a:        AppendLong(nil, 1),
			b:        AppendLong(nil, 2),
			expected: -1,
		},
		{
			name:     "larger positive first",
			a:        AppendLong(nil, 2),
			b:        AppendLong(nil, 1),
			expected: 1,
		},
		{
			name:     "negative before zero",
			a:        AppendLong(nil, -1),
			b:        AppendLong(nil, 0),
			expected: -1,
		},
		{
			name:     "negative sorts before a positive with a smaller zigzag encoding",
			a:        AppendLong(nil, -2),
			b:        AppendLong(nil, 1),
			expected: -1,
		},
		{
			name:     "more negative first",
			a:        AppendLong(nil, -1000),
			b:        AppendLong(nil, -999),
			expected: -1,
		},
		{
			name:     "equal negatives",
			a:        AppendLong(nil, -64),
			b:        AppendLong(nil, -64),
			expected: 0,
		},
		{
			name:     "one-byte varint against multi-byte varint",
			a:        AppendLong(nil, 63),
			b:        AppendLong(nil, 64),
			expected: -1,
		},
		{
			name:     "multi-byte varints",
			a:        AppendLong(nil, 1<<40),
			b:        AppendLong(nil, 1<<35),
			expected: 1,
		},
		{
			name:     "min int64 against max int64",
			a:        AppendLong(nil, math.MinInt64),
			b:        AppendLong(nil, math.MaxInt64),
			expected: -1,
		},
		{
			name:     "max int64 against min int64",
			a:        AppendLong(nil, math.MaxInt64),
			b:        AppendLong(nil, math.MinInt64),
			expected: 1,
		},
		{
			name:     "overlong zero equals zero",
			a:        []byte{0x80, 0x00},
			b:        []byte{0x00},
			expected: 0,
		},
		{
			name:     "overlong encoding compares by value",
			a:        overlong(-65, 3),
			b:        AppendLong(nil, -65),
			expected: 0,
		},
		{
			name:     "overlong encoding of a smaller value",
			a:        overlong(5, 4),
			b:        AppendLong(nil, 6),
			expected: -1,
		},
		{
			name:     "trailing bytes are not consumed",
			a:        append(AppendLong(nil, 300), 0xff, 0xff),
			b:        append(AppendLong(nil, 300), 0x01),
			expected: 0,
		},
		{
			name: "empty a",
			a:    nil,
			b:    AppendLong(nil, 1),
		},
		{
			name: "empty b",
			a:    AppendLong(nil, 1),
			b:    []byte{},
		},
		{
			name: "both empty",
			a:    nil,
			b:    nil,
		},
		{
			name: "a truncated mid-varint",
			a:    AppendLong(nil, math.MaxInt64)[:4],
			b:    AppendLong(nil, 1),
		},
		{
			name: "b truncated mid-varint",
			a:    AppendLong(nil, math.MinInt64),
			b:    []byte{0x80},
		},
		{
			name: "a overflows",
			a:    tenthByteOverflow,
			b:    AppendLong(nil, 0),
		},
		{
			name: "b overflows",
			a:    AppendLong(nil, 0),
			b:    tenthByteOverflow,
		},
		{
			name: "a overflows and b truncated",
			a:    tenthByteOverflow,
			b:    []byte{0xff},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, na, nb := CompareLong(tc.a, tc.b)

			da := decodeWith(tc.a, (*BinaryReader).ReadLong)
			db := decodeWith(tc.b, (*BinaryReader).ReadLong)
			requireCompare(t, da, db, cmp.Compare[int64], c, na, nb)
			require.Equal(t, tc.expected, c)
		})
	}
}

func TestCompareBytes(t *testing.T) {
	t.Parallel()

	bytesOf := func(b ...byte) []byte { return AppendBytes(nil, b) }
	long := bytes.Repeat([]byte{0x5a}, 200)
	longer := append(bytes.Clone(long), 0x00)
	overlongLength := append(overlong(3, 2), 'a', 'b', 'c')
	negativeLength := AppendLong(nil, -1)
	oversizeLength := AppendLong(nil, math.MaxInt32+1)

	testCases := []struct {
		name     string
		a        []byte
		b        []byte
		expected int
	}{
		{
			name:     "both empty values",
			a:        bytesOf(),
			b:        bytesOf(),
			expected: 0,
		},
		{
			name:     "equal values",
			a:        AppendString(nil, "avro"),
			b:        AppendString(nil, "avro"),
			expected: 0,
		},
		{
			name:     "first byte decides",
			a:        AppendString(nil, "apple"),
			b:        AppendString(nil, "banana"),
			expected: -1,
		},
		{
			name:     "later byte decides",
			a:        AppendString(nil, "abd"),
			b:        AppendString(nil, "abc"),
			expected: 1,
		},
		{
			name:     "bytes compare unsigned",
			a:        bytesOf(0x7f),
			b:        bytesOf(0x80),
			expected: -1,
		},
		{
			name:     "high byte sorts after low byte",
			a:        bytesOf(0xff),
			b:        bytesOf(0x00),
			expected: 1,
		},
		{
			name:     "empty value is a prefix of every value",
			a:        bytesOf(),
			b:        bytesOf(0x00),
			expected: -1,
		},
		{
			name:     "prefix sorts first",
			a:        AppendString(nil, "avro"),
			b:        AppendString(nil, "avro-go"),
			expected: -1,
		},
		{
			name:     "longer value with a prefix sorts after",
			a:        AppendString(nil, "avro-go"),
			b:        AppendString(nil, "avro"),
			expected: 1,
		},
		{
			name:     "shorter value sorts after a longer one it exceeds",
			a:        AppendString(nil, "b"),
			b:        AppendString(nil, "abc"),
			expected: 1,
		},
		{
			name:     "multi-byte length prefixes",
			a:        AppendBytes(nil, long),
			b:        AppendBytes(nil, longer),
			expected: -1,
		},
		{
			name:     "overlong length prefix compares by payload",
			a:        overlongLength,
			b:        AppendString(nil, "abc"),
			expected: 0,
		},
		{
			name:     "trailing bytes are not consumed",
			a:        append(AppendString(nil, "x"), 'y'),
			b:        append(AppendString(nil, "x"), 'a', 'b'),
			expected: 0,
		},
		{
			name: "empty a",
			a:    nil,
			b:    bytesOf(0x01),
		},
		{
			name: "empty b",
			a:    bytesOf(0x01),
			b:    []byte{},
		},
		{
			name: "both empty inputs",
			a:    nil,
			b:    nil,
		},
		{
			name: "a truncated in its length prefix",
			a:    AppendBytes(nil, long)[:1],
			b:    bytesOf(0x01),
		},
		{
			name: "b truncated in its payload",
			a:    AppendString(nil, "abc"),
			b:    AppendString(nil, "abc")[:3],
		},
		{
			name: "a truncated with only its length prefix",
			a:    []byte{0x04},
			b:    bytesOf(),
		},
		{
			name: "a with a negative length",
			a:    negativeLength,
			b:    bytesOf(0x01),
		},
		{
			name: "b with a length above max int32",
			a:    bytesOf(0x01),
			b:    oversizeLength,
		},
		{
			name: "a with a negative length and b truncated",
			a:    negativeLength,
			b:    []byte{0x06, 'a'},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, na, nb := CompareBytes(tc.a, tc.b)

			da := decodeWith(tc.a, (*BinaryReader).ReadBytes)
			db := decodeWith(tc.b, (*BinaryReader).ReadBytes)
			requireCompare(t, da, db, bytes.Compare, c, na, nb)
			require.Equal(t, tc.expected, c)
		})
	}
}

func TestCompareBytes_readsOnlyThePayload(t *testing.T) {
	t.Parallel()

	// A comparison reads each payload and nothing past it: the trailing bytes
	// differ but do not affect the result, and a changed payload byte does.
	// That the payloads are not copied rests on TestCompareAllocations.
	a := append(AppendString(nil, "abc"), 0xee)
	b := append(AppendString(nil, "abc"), 0x11)

	c, na, nb := CompareBytes(a, b)
	require.Equal(t, 0, c)
	require.Equal(t, len(a)-1, na)
	require.Equal(t, len(b)-1, nb)

	a[len(a)-2] = 'd'
	c, _, _ = CompareBytes(a, b)
	require.Equal(t, 1, c)
}

var compareSinkC, compareSinkNA, compareSinkNB int

func TestCompareAllocations(t *testing.T) {
	// Not parallel: allocation counts are only meaningful when nothing else in
	// this test binary is allocating at the same time.

	minLong := AppendLong(nil, math.MinInt64)
	maxLong := AppendLong(nil, math.MaxInt64)
	payloadA := AppendBytes(nil, bytes.Repeat([]byte{0xab}, 64))
	payloadB := AppendBytes(nil, bytes.Repeat([]byte{0xab}, 65))

	testCases := []struct {
		name string
		f    func()
	}{
		{name: "CompareLong", f: func() {
			compareSinkC, compareSinkNA, compareSinkNB = CompareLong(minLong, maxLong)
		}},
		{name: "CompareLong truncated", f: func() {
			compareSinkC, compareSinkNA, compareSinkNB = CompareLong(minLong[:3], maxLong)
		}},
		{name: "CompareBytes", f: func() {
			compareSinkC, compareSinkNA, compareSinkNB = CompareBytes(payloadA, payloadB)
		}},
		{name: "CompareBytes truncated", f: func() {
			compareSinkC, compareSinkNA, compareSinkNB = CompareBytes(payloadA, payloadB[:10])
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Zero(t, testing.AllocsPerRun(100, tc.f))
		})
	}

	// The success paths above must actually succeed, or they measure nothing.
	c, na, nb := CompareLong(minLong, maxLong)
	require.Equal(t, -1, c)
	require.Equal(t, len(minLong), na)
	require.Equal(t, len(maxLong), nb)
	c, na, nb = CompareBytes(payloadA, payloadB)
	require.Equal(t, -1, c)
	require.Equal(t, len(payloadA), na)
	require.Equal(t, len(payloadB), nb)
}
