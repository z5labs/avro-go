// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// bytesCase is one valid value, expressed through every API that encodes or
// decodes it, so that a round trip can compare the []byte functions with
// BinaryWriter and BinaryReader.
type bytesCase struct {
	name     string
	expected any
	write    func(w *BinaryWriter) error
	append   func(dst []byte) []byte
	consume  func(b []byte) (any, int)
	read     func(r *BinaryReader) (any, error)

	// size is the matching Size function, nil when the type has none.
	size func() int
}

func longCase(v int64) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("long %d", v),
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteLong(v) },
		append:   func(dst []byte) []byte { return AppendLong(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeLong(b) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadLong() },
		size:     func() int { return SizeLong(v) },
	}
}

func intCase(v int32) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("int %d", v),
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteInt(v) },
		append:   func(dst []byte) []byte { return AppendInt(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeInt(b) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadInt() },
		size:     func() int { return SizeLong(int64(v)) },
	}
}

func indexCase(i, count int) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("index %d of %d", i, count),
		expected: i,
		write:    func(w *BinaryWriter) error { return w.WriteLong(int64(i)) },
		append:   func(dst []byte) []byte { return AppendIndex(dst, i) },
		consume:  func(b []byte) (any, int) { return ConsumeIndex(b, count) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadIndex(count) },
		size:     func() int { return SizeIndex(i) },
	}
}

func lengthCase(l int) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("length %d", l),
		expected: l,
		write:    func(w *BinaryWriter) error { return w.WriteLong(int64(l)) },
		append:   func(dst []byte) []byte { return AppendLong(dst, int64(l)) },
		consume:  func(b []byte) (any, int) { return ConsumeLength(b) },
		read: func(r *BinaryReader) (any, error) {
			v, err := r.ReadLong()
			return int(v), err
		},
		size: func() int { return SizeLong(int64(l)) },
	}
}

func boolCase(v bool) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("bool %t", v),
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteBool(v) },
		append:   func(dst []byte) []byte { return AppendBool(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeBool(b) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadBool() },
	}
}

func floatCase(v float32) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("float %v", v),
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteFloat(v) },
		append:   func(dst []byte) []byte { return AppendFloat(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeFloat(b) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadFloat() },
	}
}

func doubleCase(v float64) bytesCase {
	return bytesCase{
		name:     fmt.Sprintf("double %v", v),
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteDouble(v) },
		append:   func(dst []byte) []byte { return AppendDouble(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeDouble(b) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadDouble() },
	}
}

func bytesValueCase(name string, v []byte) bytesCase {
	return bytesCase{
		name:     "bytes " + name,
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteBytes(v) },
		append:   func(dst []byte) []byte { return AppendBytes(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeBytes(b) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadBytes() },
		size:     func() int { return SizeBytes(len(v)) },
	}
}

func stringCase(name string, v string) bytesCase {
	return bytesCase{
		name:     "string " + name,
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteString(v) },
		append:   func(dst []byte) []byte { return AppendString(dst, v) },
		consume: func(b []byte) (any, int) {
			s, n := ConsumeBytes(b)
			return string(s), n
		},
		read: func(r *BinaryReader) (any, error) { return r.ReadString() },
		size: func() int { return SizeBytes(len(v)) },
	}
}

func fixedCase(name string, v []byte) bytesCase {
	return bytesCase{
		name:     "fixed " + name,
		expected: v,
		write:    func(w *BinaryWriter) error { return w.WriteFixed(v) },
		append:   func(dst []byte) []byte { return AppendFixed(dst, v) },
		consume:  func(b []byte) (any, int) { return ConsumeFixed(b, len(v)) },
		read:     func(r *BinaryReader) (any, error) { return r.ReadFixed(len(v)) },
		size:     func() int { return len(v) },
	}
}

func validBytesCases() []bytesCase {
	return []bytesCase{
		longCase(0),
		longCase(1),
		longCase(-1),
		longCase(63),
		longCase(-64),
		longCase(64),
		longCase(-65),
		longCase(math.MaxInt32),
		longCase(math.MinInt32),
		longCase(math.MaxInt64),
		longCase(math.MinInt64),
		intCase(0),
		intCase(-1),
		intCase(64),
		intCase(math.MaxInt32),
		intCase(math.MinInt32),
		indexCase(0, 1),
		indexCase(2, 3),
		indexCase(200, 201),
		lengthCase(0),
		lengthCase(64),
		lengthCase(math.MaxInt32),
		boolCase(false),
		boolCase(true),
		floatCase(0),
		floatCase(3.14),
		floatCase(float32(math.Inf(-1))),
		floatCase(math.MaxFloat32),
		doubleCase(0),
		doubleCase(-2.718281828459045),
		doubleCase(math.Inf(1)),
		doubleCase(math.SmallestNonzeroFloat64),
		bytesValueCase("empty", []byte{}),
		bytesValueCase("short", []byte{0x00, 0xff, 0x7f}),
		bytesValueCase("with a two byte length prefix", bytes.Repeat([]byte{0xab}, 64)),
		stringCase("empty", ""),
		stringCase("ascii", "hello, avro"),
		stringCase("utf-8", "héllo, 世界"),
		fixedCase("empty", []byte{}),
		fixedCase("sixteen bytes", bytes.Repeat([]byte{0x5a}, 16)),
	}
}

func TestBytesRoundTrip(t *testing.T) {
	t.Parallel()

	for _, tc := range validBytesCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			require.NoError(t, tc.write(NewBinaryWriter(&buf)))
			// An empty Buffer reports nil, which require.Equal tells apart
			// from the empty slice an Append function returns.
			written := append([]byte{}, buf.Bytes()...)

			prefix := []byte{0xde, 0xad}
			appended := tc.append(bytes.Clone(prefix))
			require.Equal(t, prefix, appended[:len(prefix)])
			require.Equal(t, written, appended[len(prefix):])

			if tc.size != nil {
				require.Equal(t, len(written), tc.size())
			}

			// Trailing bytes after the value must be left unconsumed.
			input := append(bytes.Clone(written), 0x02, 0x04)
			got, n := tc.consume(input)
			require.NoError(t, ConsumeError(n))
			require.Equal(t, len(written), n)
			require.Equal(t, tc.expected, got)

			read, err := tc.read(NewBinaryReader(bytes.NewReader(written)))
			require.NoError(t, err)
			require.Equal(t, read, got)
		})
	}
}

func TestConsume_truncated(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		input   []byte
		consume func(b []byte) (any, int)
	}
	var testCases []testCase
	for _, bc := range validBytesCases() {
		encoded := bc.append(nil)
		for _, prefix := range truncations(encoded) {
			testCases = append(testCases, testCase{
				name:    fmt.Sprintf("%s cut after %d of %d bytes", bc.name, len(prefix), len(encoded)),
				input:   prefix,
				consume: bc.consume,
			})
		}
	}
	testCases = append(testCases,
		testCase{
			name:    "long from nil input",
			input:   nil,
			consume: func(b []byte) (any, int) { return ConsumeLong(b) },
		},
		testCase{
			name:    "long with a continuation byte and nothing after it",
			input:   []byte{0x80},
			consume: func(b []byte) (any, int) { return ConsumeLong(b) },
		},
		testCase{
			name:    "bytes with a length prefix and no payload",
			input:   []byte{0x04},
			consume: func(b []byte) (any, int) { return ConsumeBytes(b) },
		},
		testCase{
			name:    "fixed one byte short",
			input:   []byte{0x01, 0x02, 0x03},
			consume: func(b []byte) (any, int) { return ConsumeFixed(b, 4) },
		},
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, n := tc.consume(tc.input)

			require.Negative(t, n)
			require.Equal(t, io.ErrUnexpectedEOF, ConsumeError(n))
		})
	}
}

func TestConsume_corrupt(t *testing.T) {
	t.Parallel()

	tenContinuationBytes := bytes.Repeat([]byte{0x80}, 10)
	tenthByte := func(b byte) []byte {
		return append(bytes.Repeat([]byte{0xff}, 9), b)
	}

	testCases := []struct {
		name        string
		input       []byte
		consume     func(b []byte) int
		expectedErr error
	}{
		{
			name:        "long with a 10th byte of 0x02",
			input:       tenthByte(0x02),
			consume:     func(b []byte) int { _, n := ConsumeLong(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "long with a 10th byte of 0x7f",
			input:       tenthByte(0x7f),
			consume:     func(b []byte) int { _, n := ConsumeLong(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "long with 10 continuation bytes",
			input:       tenContinuationBytes,
			consume:     func(b []byte) int { _, n := ConsumeLong(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "long with 11 bytes",
			input:       append(bytes.Clone(tenContinuationBytes), 0x01),
			consume:     func(b []byte) int { _, n := ConsumeLong(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "int above max int32",
			input:       AppendLong(nil, math.MaxInt32+1),
			consume:     func(b []byte) int { _, n := ConsumeInt(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "int below min int32",
			input:       AppendLong(nil, math.MinInt32-1),
			consume:     func(b []byte) int { _, n := ConsumeInt(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "int whose long overflows",
			input:       tenthByte(0x02),
			consume:     func(b []byte) int { _, n := ConsumeInt(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "length that is negative",
			input:       AppendLong(nil, -1),
			consume:     func(b []byte) int { _, n := ConsumeLength(b); return n },
			expectedErr: ErrNegativeLength,
		},
		{
			name:        "length above max int32",
			input:       AppendLong(nil, math.MaxInt32+1),
			consume:     func(b []byte) int { _, n := ConsumeLength(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "length whose long overflows",
			input:       tenthByte(0x02),
			consume:     func(b []byte) int { _, n := ConsumeLength(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "index that is negative",
			input:       AppendLong(nil, -1),
			consume:     func(b []byte) int { _, n := ConsumeIndex(b, 3); return n },
			expectedErr: ErrInvalidIndex,
		},
		{
			name:        "index equal to count",
			input:       AppendLong(nil, 3),
			consume:     func(b []byte) int { _, n := ConsumeIndex(b, 3); return n },
			expectedErr: ErrInvalidIndex,
		},
		{
			name:        "index above count",
			input:       AppendLong(nil, math.MaxInt64),
			consume:     func(b []byte) int { _, n := ConsumeIndex(b, 3); return n },
			expectedErr: ErrInvalidIndex,
		},
		{
			name:        "index of a zero count",
			input:       AppendLong(nil, 0),
			consume:     func(b []byte) int { _, n := ConsumeIndex(b, 0); return n },
			expectedErr: ErrInvalidIndex,
		},
		{
			name:        "index whose long overflows",
			input:       tenthByte(0x02),
			consume:     func(b []byte) int { _, n := ConsumeIndex(b, 3); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "bytes with a negative length",
			input:       []byte{0x01, 0xaa},
			consume:     func(b []byte) int { _, n := ConsumeBytes(b); return n },
			expectedErr: ErrNegativeLength,
		},
		{
			name:        "bytes with a length above max int32",
			input:       AppendLong(nil, math.MaxInt32+1),
			consume:     func(b []byte) int { _, n := ConsumeBytes(b); return n },
			expectedErr: ErrOverflow,
		},
		{
			name:        "bytes whose length overflows",
			input:       tenthByte(0x02),
			consume:     func(b []byte) int { _, n := ConsumeBytes(b); return n },
			expectedErr: ErrOverflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			n := tc.consume(tc.input)

			require.Negative(t, n)
			require.Equal(t, tc.expectedErr, ConsumeError(n))
		})
	}
}

func TestConsumeLong_overlong(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     []byte
		expected  int64
		expectedN int
	}{
		{
			name:      "zero in two bytes",
			input:     []byte{0x80, 0x00},
			expected:  0,
			expectedN: 2,
		},
		{
			name:      "negative one in three bytes",
			input:     []byte{0x81, 0x80, 0x00},
			expected:  -1,
			expectedN: 3,
		},
		{
			name:      "zero in ten bytes",
			input:     append(bytes.Repeat([]byte{0x80}, 9), 0x00),
			expected:  0,
			expectedN: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, n := ConsumeLong(tc.input)

			require.Equal(t, tc.expectedN, n)
			require.Equal(t, tc.expected, got)

			read, err := NewBinaryReader(bytes.NewReader(tc.input)).ReadLong()
			require.NoError(t, err)
			require.Equal(t, read, got)
		})
	}
}

func TestConsumeBool_nonZeroIsTrue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    byte
		expected bool
	}{
		{name: "zero", input: 0x00, expected: false},
		{name: "one", input: 0x01, expected: true},
		{name: "two", input: 0x02, expected: true},
		{name: "high bit", input: 0x80, expected: true},
		{name: "all bits", input: 0xff, expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, n := ConsumeBool([]byte{tc.input})

			require.Equal(t, 1, n)
			require.Equal(t, tc.expected, got)

			read, err := NewBinaryReader(bytes.NewReader([]byte{tc.input})).ReadBool()
			require.NoError(t, err)
			require.Equal(t, read, got)
		})
	}
}

func TestConsumeBytes_aliasing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   []byte
		consume func(b []byte) ([]byte, int)
		offset  int
	}{
		{
			name:    "bytes",
			input:   []byte{0x06, 'a', 'b', 'c', 'd', 'e'},
			consume: ConsumeBytes,
			offset:  1,
		},
		{
			name:    "fixed",
			input:   []byte{'a', 'b', 'c', 'd', 'e'},
			consume: func(b []byte) ([]byte, int) { return ConsumeFixed(b, 3) },
			offset:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, n := tc.consume(tc.input)
			require.Equal(t, tc.offset+3, n)
			require.Equal(t, []byte("abc"), got)
			require.Equal(t, len(got), cap(got))
			require.Same(t, &tc.input[tc.offset], &got[0])

			// Appending must copy rather than overwrite what follows in b.
			_ = append(got, 'x')
			require.Equal(t, byte('d'), tc.input[tc.offset+3])

			// Overwriting b is visible through the result.
			tc.input[tc.offset] = 'z'
			require.Equal(t, byte('z'), got[0])
		})
	}
}

func TestConsumeBytes_emptyIsNonNil(t *testing.T) {
	t.Parallel()

	got, n := ConsumeBytes([]byte{0x00, 0xff})

	require.Equal(t, 1, n)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.Equal(t, 0, cap(got))
}

func TestConsumeFixed_size(t *testing.T) {
	t.Parallel()

	t.Run("zero size succeeds without consuming", func(t *testing.T) {
		t.Parallel()

		got, n := ConsumeFixed([]byte{0x01}, 0)

		require.Equal(t, 0, n)
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("zero size of nil input is nil", func(t *testing.T) {
		t.Parallel()

		got, n := ConsumeFixed(nil, 0)

		require.Equal(t, 0, n)
		require.Nil(t, got)
	})

	t.Run("negative size panics", func(t *testing.T) {
		t.Parallel()

		require.Panics(t, func() {
			ConsumeFixed([]byte{0x01}, -1)
		})
	})
}

func TestConsumeError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		n        int
		expected error
	}{
		{name: "zero is success", n: 0, expected: nil},
		{name: "positive is success", n: 10, expected: nil},
		{name: "truncated", n: errCodeTruncated, expected: io.ErrUnexpectedEOF},
		{name: "overflow", n: errCodeOverflow, expected: ErrOverflow},
		{name: "negative length", n: errCodeNegativeLength, expected: ErrNegativeLength},
		{name: "invalid index", n: errCodeInvalidIndex, expected: ErrInvalidIndex},
		{name: "unknown code", n: math.MinInt, expected: io.ErrUnexpectedEOF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, ConsumeError(tc.n))
		})
	}
}

func TestSizeLong(t *testing.T) {
	t.Parallel()

	var values []int64
	for shift := range 63 {
		v := int64(1) << shift
		values = append(values, v-1, v, -v, -v-1)
	}
	values = append(values, math.MaxInt64, math.MinInt64)

	for _, v := range values {
		t.Run(fmt.Sprintf("%d", v), func(t *testing.T) {
			t.Parallel()

			require.Equal(t, len(AppendLong(nil, v)), SizeLong(v))
		})
	}
}

// bytesSink keeps the results of the allocation tests and benchmarks live.
var (
	bytesSinkLong  int64
	bytesSinkInt   int32
	bytesSinkN     int
	bytesSinkBool  bool
	bytesSinkF32   float32
	bytesSinkF64   float64
	bytesSinkBytes []byte
	bytesSinkErr   error
)

func TestBytesAllocations(t *testing.T) {
	// Not parallel: allocation counts are only meaningful when nothing else in
	// this test binary is allocating at the same time.

	long := AppendLong(nil, math.MinInt64)
	index := AppendIndex(nil, 200)
	payload := AppendBytes(nil, bytes.Repeat([]byte{0xab}, 64))
	double := AppendDouble(nil, math.Pi)
	value := bytes.Repeat([]byte{0xcd}, 64)
	dst := make([]byte, 0, 128)

	testCases := []struct {
		name string
		f    func()
	}{
		{name: "ConsumeLong", f: func() { bytesSinkLong, bytesSinkN = ConsumeLong(long) }},
		{name: "ConsumeInt", f: func() { bytesSinkInt, bytesSinkN = ConsumeInt(index) }},
		{name: "ConsumeInt overflowing", f: func() { bytesSinkInt, bytesSinkN = ConsumeInt(long) }},
		{name: "ConsumeLength", f: func() { bytesSinkN, bytesSinkN = ConsumeLength(payload) }},
		{name: "ConsumeIndex", f: func() { bytesSinkN, bytesSinkN = ConsumeIndex(index, 201) }},
		{name: "ConsumeIndex out of range", f: func() { bytesSinkN, bytesSinkN = ConsumeIndex(long, 1) }},
		{name: "ConsumeBytes", f: func() { bytesSinkBytes, bytesSinkN = ConsumeBytes(payload) }},
		{name: "ConsumeFixed", f: func() { bytesSinkBytes, bytesSinkN = ConsumeFixed(payload, 16) }},
		{name: "ConsumeBool", f: func() { bytesSinkBool, bytesSinkN = ConsumeBool(long) }},
		{name: "ConsumeFloat", f: func() { bytesSinkF32, bytesSinkN = ConsumeFloat(double) }},
		{name: "ConsumeDouble", f: func() { bytesSinkF64, bytesSinkN = ConsumeDouble(double) }},
		{name: "ConsumeError", f: func() {
			_, n := ConsumeLong(nil)
			bytesSinkErr = ConsumeError(n)
		}},
		{name: "AppendLong", f: func() { bytesSinkBytes = AppendLong(dst[:0], math.MinInt64) }},
		{name: "AppendInt", f: func() { bytesSinkBytes = AppendInt(dst[:0], math.MinInt32) }},
		{name: "AppendIndex", f: func() { bytesSinkBytes = AppendIndex(dst[:0], 200) }},
		{name: "AppendBool", f: func() { bytesSinkBytes = AppendBool(dst[:0], true) }},
		{name: "AppendFloat", f: func() { bytesSinkBytes = AppendFloat(dst[:0], math.E) }},
		{name: "AppendDouble", f: func() { bytesSinkBytes = AppendDouble(dst[:0], math.Pi) }},
		{name: "AppendBytes", f: func() { bytesSinkBytes = AppendBytes(dst[:0], value) }},
		{name: "AppendString", f: func() { bytesSinkBytes = AppendString(dst[:0], "hello, avro") }},
		{name: "AppendFixed", f: func() { bytesSinkBytes = AppendFixed(dst[:0], value) }},
		{name: "SizeLong", f: func() { bytesSinkN = SizeLong(math.MinInt64) }},
		{name: "SizeIndex", f: func() { bytesSinkN = SizeIndex(200) }},
		{name: "SizeBytes", f: func() { bytesSinkN = SizeBytes(64) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Zero(t, testing.AllocsPerRun(100, tc.f))
		})
	}

	// The success paths above must actually succeed, or they measure nothing.
	_, n := ConsumeInt(index)
	require.Equal(t, len(index), n)
	_, n = ConsumeIndex(index, 201)
	require.Equal(t, len(index), n)
}

func BenchmarkConsumeLong(b *testing.B) {
	for _, v := range []int64{1, math.MaxInt32, math.MinInt64} {
		buf := AppendLong(nil, v)
		b.Run(fmt.Sprintf("%d bytes", len(buf)), func(b *testing.B) {
			for b.Loop() {
				bytesSinkLong, bytesSinkN = ConsumeLong(buf)
			}
		})
	}
}

func BenchmarkConsumeFixed(b *testing.B) {
	buf := bytes.Repeat([]byte{0xab}, 16)
	for b.Loop() {
		bytesSinkBytes, bytesSinkN = ConsumeFixed(buf, 16)
	}
}

func BenchmarkConsumeBytes(b *testing.B) {
	buf := AppendBytes(nil, bytes.Repeat([]byte{0xab}, 64))
	for b.Loop() {
		bytesSinkBytes, bytesSinkN = ConsumeBytes(buf)
	}
}

func BenchmarkAppendLong(b *testing.B) {
	dst := make([]byte, 0, 16)
	for _, v := range []int64{1, math.MaxInt32, math.MinInt64} {
		b.Run(fmt.Sprintf("%d bytes", SizeLong(v)), func(b *testing.B) {
			for b.Loop() {
				bytesSinkBytes = AppendLong(dst[:0], v)
			}
		})
	}
}

func BenchmarkAppendBytes(b *testing.B) {
	dst := make([]byte, 0, 128)
	value := bytes.Repeat([]byte{0xab}, 64)
	for b.Loop() {
		bytesSinkBytes = AppendBytes(dst[:0], value)
	}
}

func BenchmarkAppendDouble(b *testing.B) {
	dst := make([]byte, 0, 16)
	for b.Loop() {
		bytesSinkBytes = AppendDouble(dst[:0], math.Pi)
	}
}

func BenchmarkSizeLong(b *testing.B) {
	for _, v := range []int64{1, math.MaxInt32, math.MinInt64} {
		b.Run(fmt.Sprintf("%d bytes", SizeLong(v)), func(b *testing.B) {
			for b.Loop() {
				bytesSinkN = SizeLong(v)
			}
		})
	}
}

// The remaining Append benchmarks call each function directly in the loop,
// rather than through a closure, so that they measure it inlined.
func BenchmarkAppendInt(b *testing.B) {
	dst := make([]byte, 0, 16)
	for b.Loop() {
		bytesSinkBytes = AppendInt(dst[:0], math.MinInt32)
	}
}

func BenchmarkAppendIndex(b *testing.B) {
	dst := make([]byte, 0, 16)
	for b.Loop() {
		bytesSinkBytes = AppendIndex(dst[:0], 200)
	}
}

func BenchmarkAppendBool(b *testing.B) {
	dst := make([]byte, 0, 16)
	for b.Loop() {
		bytesSinkBytes = AppendBool(dst[:0], true)
	}
}

func BenchmarkAppendFloat(b *testing.B) {
	dst := make([]byte, 0, 16)
	for b.Loop() {
		bytesSinkBytes = AppendFloat(dst[:0], math.E)
	}
}

func BenchmarkAppendString(b *testing.B) {
	dst := make([]byte, 0, 128)
	for b.Loop() {
		bytesSinkBytes = AppendString(dst[:0], "hello, avro")
	}
}

func BenchmarkAppendFixed(b *testing.B) {
	dst := make([]byte, 0, 128)
	value := bytes.Repeat([]byte{0xab}, 16)
	for b.Loop() {
		bytesSinkBytes = AppendFixed(dst[:0], value)
	}
}

func BenchmarkSizeIndex(b *testing.B) {
	for b.Loop() {
		bytesSinkN = SizeIndex(200)
	}
}

func BenchmarkSizeBytes(b *testing.B) {
	for b.Loop() {
		bytesSinkN = SizeBytes(64)
	}
}
