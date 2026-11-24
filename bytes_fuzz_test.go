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

// The fuzz targets in this file decode the same input through the []byte API,
// a bytes-mode BinaryReader and the Consume functions, and through an
// io.Reader-mode BinaryReader, and require that they agree. The io.Reader-mode
// reader is the reference: it is the older of the two decoders, and the one
// every other is documented against.
//
// Their seed corpora run as ordinary tests under go test. To fuzz one, run for
// example:
//
//	go test -run '^$' -fuzz '^FuzzBytesReader$' -fuzztime 1m .

// fuzzReadKind selects a BinaryReader read method.
type fuzzReadKind uint8

const (
	fuzzReadBool fuzzReadKind = iota
	fuzzReadInt
	fuzzReadLong
	fuzzReadIndex
	fuzzReadFloat
	fuzzReadDouble
	fuzzReadBytes
	fuzzReadString
	fuzzReadFixed

	// fuzzReadKinds is the number of read methods; a fuzzed kind is taken
	// modulo it, so that every byte selects some method.
	fuzzReadKinds
)

// maxFuzzReaderAlloc bounds what an io.Reader-mode read may allocate for a
// declared length that runs past the end of the input. See
// fuzzRead.overAllocates.
const maxFuzzReaderAlloc = 1 << 16

// fuzzRead is one read: a method, and the count of an index or size of a
// fixed it is called with.
type fuzzRead struct {
	kind fuzzReadKind
	arg  int
}

func newFuzzRead(kind uint8, arg uint16) fuzzRead {
	return fuzzRead{kind: fuzzReadKind(kind % uint8(fuzzReadKinds)), arg: int(arg)}
}

// fuzzValue normalises a decoded value so that require.Equal compares what was
// decoded rather than how it is represented: floats by their bits, so that NaN
// equals itself, and []byte by its contents, so that an empty fixed decoded
// from nil input equals one decoded from an empty slice.
func fuzzValue(v any) any {
	switch v := v.(type) {
	case float32:
		return math.Float32bits(v)
	case float64:
		return math.Float64bits(v)
	case []byte:
		return string(v)
	default:
		return v
	}
}

// read calls the read method on r.
func (op fuzzRead) read(r *BinaryReader) (any, error) {
	var v any
	var err error
	switch op.kind {
	case fuzzReadBool:
		v, err = r.ReadBool()
	case fuzzReadInt:
		v, err = r.ReadInt()
	case fuzzReadLong:
		v, err = r.ReadLong()
	case fuzzReadIndex:
		v, err = r.ReadIndex(op.arg)
	case fuzzReadFloat:
		v, err = r.ReadFloat()
	case fuzzReadDouble:
		v, err = r.ReadDouble()
	case fuzzReadBytes:
		v, err = r.ReadBytes()
	case fuzzReadString:
		v, err = r.ReadString()
	case fuzzReadFixed:
		v, err = r.ReadFixed(op.arg)
	default:
		panic("fuzzRead: unknown kind")
	}
	return fuzzValue(v), err
}

// consume calls the Consume function matching the read method on b.
func (op fuzzRead) consume(b []byte) (any, int) {
	var v any
	var n int
	switch op.kind {
	case fuzzReadBool:
		v, n = ConsumeBool(b)
	case fuzzReadInt:
		v, n = ConsumeInt(b)
	case fuzzReadLong:
		v, n = ConsumeLong(b)
	case fuzzReadIndex:
		v, n = ConsumeIndex(b, op.arg)
	case fuzzReadFloat:
		v, n = ConsumeFloat(b)
	case fuzzReadDouble:
		v, n = ConsumeDouble(b)
	case fuzzReadBytes:
		v, n = ConsumeBytes(b)
	case fuzzReadString:
		var s []byte
		s, n = ConsumeBytes(b)
		v = string(s)
	case fuzzReadFixed:
		v, n = ConsumeFixed(b, op.arg)
	default:
		panic("fuzzRead: unknown kind")
	}
	return fuzzValue(v), n
}

// overAllocates reports whether an io.Reader-mode read of op from rest would
// allocate a buffer far larger than rest before finding that the input ends.
//
// ReadBytes and ReadString in io.Reader mode allocate the declared length up
// front, so a fuzzed length prefix just under math.MaxInt32 costs 2 GiB per
// read. Such a read is cut short whatever happens, so the fuzz targets skip the
// io.Reader-mode read for it and require io.ErrUnexpectedEOF of the others.
// The length is decoded with encoding/binary rather than ConsumeLength, so the
// code under test does not decide which of its own inputs are compared.
func (op fuzzRead) overAllocates(rest []byte) bool {
	if op.kind != fuzzReadBytes && op.kind != fuzzReadString {
		return false
	}
	l, n := binary.Varint(rest)
	return n > 0 && l > maxFuzzReaderAlloc && l <= math.MaxInt32 && l > int64(len(rest)-n)
}

// fuzzSentinels are the errors a read can fail with, in the order fuzzSentinel
// tries them.
var fuzzSentinels = []error{io.EOF, io.ErrUnexpectedEOF, ErrOverflow, ErrNegativeLength, ErrInvalidIndex}

// fuzzSentinel returns the sentinel err matches with errors.Is, failing t when
// it matches none.
func fuzzSentinel(t *testing.T, err error) error {
	t.Helper()

	for _, s := range fuzzSentinels {
		if errors.Is(err, s) {
			return s
		}
	}
	require.Failf(t, "error matches no sentinel", "%T: %v", err, err)
	return nil
}

// consumeSentinel is the sentinel a Consume function reports where the
// io.Reader-mode reader reported want: the same one, except that a Consume
// function reports every truncation as io.ErrUnexpectedEOF.
func consumeSentinel(want error) error {
	if want == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return want
}

// requireReaderErr asserts that err is a *BinaryReaderError and returns the
// sentinel it wraps.
func requireReaderErr(t *testing.T, err error) error {
	t.Helper()

	var rerr *BinaryReaderError
	require.True(t, errors.As(err, &rerr), "expected a *BinaryReaderError, got %T: %v", err, err)
	return fuzzSentinel(t, err)
}

// requireSameRead asserts that a bytes-mode read agreed with an io.Reader-mode
// one: the same value on success, and the same sentinel on failure.
func requireSameRead(t *testing.T, want any, wantErr error, got any, gotErr error) {
	t.Helper()

	if wantErr != nil {
		require.Error(t, gotErr, "io.Reader mode failed with %v, bytes mode decoded %v", wantErr, got)
		require.Equal(t, requireReaderErr(t, wantErr), requireReaderErr(t, gotErr))
		return
	}
	require.NoError(t, gotErr, "io.Reader mode decoded %v", want)
	require.Equal(t, want, got)
}

// requireSameConsume asserts that a Consume function agreed with an
// io.Reader-mode read, which consumed offset bytes on success.
func requireSameConsume(t *testing.T, want any, wantErr error, offset int64, got any, n int) {
	t.Helper()

	if wantErr != nil {
		require.Negative(t, n, "io.Reader mode failed with %v, Consume decoded %v", wantErr, got)
		require.ErrorIs(t, ConsumeError(n), consumeSentinel(requireReaderErr(t, wantErr)))
		require.Zero(t, got)
		return
	}
	require.NoError(t, ConsumeError(n), "io.Reader mode decoded %v", want)
	require.Equal(t, want, got)
	require.Equal(t, offset, int64(n))
}

// fuzzReadSeed is one seed of FuzzBytesReader and FuzzConsume.
type fuzzReadSeed struct {
	op   fuzzRead
	data []byte
}

// fuzzVarintKinds are the read methods that begin with a varint.
var fuzzVarintKinds = []fuzzReadKind{fuzzReadInt, fuzzReadLong, fuzzReadIndex, fuzzReadBytes, fuzzReadString}

// tenthByte returns a 10-byte varint of nine continuation bytes and last.
func tenthByte(fill, last byte) []byte {
	return append(bytes.Repeat([]byte{fill}, 9), last)
}

// corruptVarints are varints, valid and not, at the edges of what a decoder
// must accept, each read by every method in fuzzVarintKinds.
func corruptVarints() [][]byte {
	return [][]byte{
		// Overlong encodings, which are accepted.
		{0x80, 0x00},
		{0x81, 0x80, 0x00},
		tenthByte(0x80, 0x00),
		// A 10th byte of 0x01 carries bit 63 and is accepted; above it
		// overflows.
		tenthByte(0xff, 0x01),
		tenthByte(0x80, 0x01),
		tenthByte(0x80, 0x02),
		tenthByte(0xff, 0x02),
		tenthByte(0x80, 0x7f),
		tenthByte(0xff, 0x7f),
		// 10 continuation bytes, and a varint continuing past them.
		bytes.Repeat([]byte{0x80}, 10),
		bytes.Repeat([]byte{0xff}, 10),
		append(bytes.Repeat([]byte{0x80}, 10), 0x01),
		// Negative lengths.
		AppendLong(nil, -1),
		append(AppendLong(nil, -1), 0xaa),
		AppendLong(nil, -64),
		AppendLong(nil, math.MinInt64),
		// Lengths above math.MaxInt32, and math.MaxInt32 itself with no
		// payload after it.
		AppendLong(nil, math.MaxInt32+1),
		AppendLong(nil, math.MaxInt64),
		AppendLong(nil, math.MaxInt32),
		// Values outside int32.
		AppendLong(nil, math.MinInt32-1),
	}
}

// fuzzReadSeeds is the seed corpus of FuzzBytesReader and FuzzConsume. Every
// seed is added whole and then cut short at every position.
func fuzzReadSeeds() []fuzzReadSeed {
	op := func(kind fuzzReadKind, arg int) fuzzRead { return fuzzRead{kind: kind, arg: arg} }

	seeds := []fuzzReadSeed{
		{op(fuzzReadBool, 0), AppendBool(nil, false)},
		{op(fuzzReadBool, 0), AppendBool(nil, true)},
		{op(fuzzReadBool, 0), []byte{0x02}},
		{op(fuzzReadInt, 0), AppendInt(nil, 0)},
		{op(fuzzReadInt, 0), AppendInt(nil, -1)},
		{op(fuzzReadInt, 0), AppendInt(nil, math.MaxInt32)},
		{op(fuzzReadInt, 0), AppendInt(nil, math.MinInt32)},
		{op(fuzzReadLong, 0), AppendLong(nil, 0)},
		{op(fuzzReadLong, 0), AppendLong(nil, 64)},
		{op(fuzzReadLong, 0), AppendLong(nil, -65)},
		{op(fuzzReadLong, 0), AppendLong(nil, math.MaxInt64)},
		{op(fuzzReadLong, 0), AppendLong(nil, math.MinInt64)},
		{op(fuzzReadIndex, 1), AppendIndex(nil, 0)},
		{op(fuzzReadIndex, 201), AppendIndex(nil, 200)},
		{op(fuzzReadFloat, 0), AppendFloat(nil, 3.14)},
		{op(fuzzReadFloat, 0), AppendFloat(nil, float32(math.NaN()))},
		{op(fuzzReadDouble, 0), AppendDouble(nil, -2.718281828459045)},
		{op(fuzzReadDouble, 0), AppendDouble(nil, math.NaN())},
		{op(fuzzReadBytes, 0), AppendBytes(nil, []byte{})},
		{op(fuzzReadBytes, 0), AppendBytes(nil, []byte{0x00, 0xff, 0x7f})},
		{op(fuzzReadBytes, 0), AppendBytes(nil, bytes.Repeat([]byte{0xab}, 64))},
		{op(fuzzReadString, 0), AppendString(nil, "")},
		{op(fuzzReadString, 0), AppendString(nil, "héllo, 世界")},
		{op(fuzzReadFixed, 0), nil},
		{op(fuzzReadFixed, 0), []byte{}},
		{op(fuzzReadFixed, 16), bytes.Repeat([]byte{0x5a}, 16)},

		// Out-of-range indexes.
		{op(fuzzReadIndex, 3), AppendLong(nil, -1)},
		{op(fuzzReadIndex, 3), AppendLong(nil, 3)},
		{op(fuzzReadIndex, 3), AppendLong(nil, 4)},
		{op(fuzzReadIndex, 0), AppendLong(nil, 0)},
		{op(fuzzReadIndex, 1), AppendLong(nil, math.MaxInt64)},
		{op(fuzzReadIndex, 1), AppendLong(nil, math.MinInt64)},
	}
	for _, v := range corruptVarints() {
		for _, kind := range fuzzVarintKinds {
			seeds = append(seeds, fuzzReadSeed{op(kind, 3), v})
		}
	}

	var all []fuzzReadSeed
	for _, s := range seeds {
		all = append(all, s)
		for _, prefix := range truncations(s.data) {
			all = append(all, fuzzReadSeed{s.op, prefix})
		}
	}
	return all
}

func addFuzzReadSeeds(f *testing.F) {
	for _, s := range fuzzReadSeeds() {
		f.Add(uint8(s.op.kind), uint16(s.op.arg), s.data)
	}
}

// FuzzBytesReader decodes one value through a bytes-mode reader and an
// io.Reader-mode reader. On success the values and Offsets must match; on
// failure the sentinels must. Error offsets are not compared: the bytes-mode
// reader consumes nothing on failure, while the io.Reader-mode reader has
// already consumed whatever it read.
func FuzzBytesReader(f *testing.F) {
	addFuzzReadSeeds(f)

	f.Fuzz(func(t *testing.T, kind uint8, arg uint16, data []byte) {
		op := newFuzzRead(kind, arg)
		br := NewBytesReader(data)

		if op.overAllocates(data) {
			_, err := op.read(br)
			require.Equal(t, io.ErrUnexpectedEOF, requireReaderErr(t, err))
			return
		}

		rr := NewBinaryReader(bytes.NewReader(data))
		want, wantErr := op.read(rr)
		got, gotErr := op.read(br)

		requireSameRead(t, want, wantErr, got, gotErr)
		if wantErr != nil {
			require.Zero(t, br.Offset())
			require.True(t, bytes.Equal(data, br.Remaining()), "a failed bytes-mode read consumed input")
			return
		}
		require.Equal(t, rr.Offset(), br.Offset())
		require.True(t, bytes.Equal(data[br.Offset():], br.Remaining()))
	})
}

// FuzzConsume decodes one value through a Consume function and an
// io.Reader-mode reader. On success the values must match and n must equal the
// reader's Offset; on failure ConsumeError(n) must match the reader's sentinel,
// except that a Consume function reports a truncation as io.ErrUnexpectedEOF
// even where the reader reports io.EOF.
func FuzzConsume(f *testing.F) {
	addFuzzReadSeeds(f)

	f.Fuzz(func(t *testing.T, kind uint8, arg uint16, data []byte) {
		op := newFuzzRead(kind, arg)

		if op.overAllocates(data) {
			_, n := op.consume(data)
			require.Equal(t, io.ErrUnexpectedEOF, ConsumeError(n))
			return
		}

		rr := NewBinaryReader(bytes.NewReader(data))
		want, wantErr := op.read(rr)
		got, n := op.consume(data)

		requireSameConsume(t, want, wantErr, rr.Offset(), got, n)
	})
}

// FuzzBytesReader_sequence runs a sequence of reads, chosen by program, over a
// bytes-mode reader, an io.Reader-mode reader and the Consume functions, all
// decoding data, so that position tracking across values is covered. Each pair
// of bytes of program is one read: its method, and the count of an index or
// size of a fixed. The sequence stops at the first read that fails, after
// which the readers' positions legitimately differ.
func FuzzBytesReader_sequence(f *testing.F) {
	type step struct {
		kind fuzzReadKind
		arg  uint8
	}
	program := func(steps ...step) []byte {
		var p []byte
		for _, s := range steps {
			p = append(p, byte(s.kind), s.arg)
		}
		return p
	}

	record := program(
		step{fuzzReadLong, 0},
		step{fuzzReadString, 0},
		step{fuzzReadIndex, 2},
		step{fuzzReadDouble, 0},
		step{fuzzReadBool, 0},
		step{fuzzReadBytes, 0},
		step{fuzzReadFixed, 4},
		step{fuzzReadFloat, 0},
		step{fuzzReadInt, 0},
		step{fuzzReadFixed, 0},
	)
	var data []byte
	data = AppendLong(data, -12345678901)
	data = AppendString(data, "héllo, 世界")
	data = AppendIndex(data, 1)
	data = AppendDouble(data, math.Pi)
	data = AppendBool(data, true)
	data = AppendBytes(data, []byte{})
	data = AppendFixed(data, []byte{0xde, 0xad, 0xbe, 0xef})
	data = AppendFloat(data, float32(math.Inf(-1)))
	data = AppendInt(data, math.MinInt32)

	f.Add(record, data)
	for _, prefix := range truncations(data) {
		f.Add(record, prefix)
	}
	// More reads than there are values, so that the last ones start at the
	// end of the input.
	f.Add(append(bytes.Clone(record), record...), data)

	// A sequence of longs that ends in each corrupt varint.
	for _, v := range corruptVarints() {
		f.Add(
			program(step{fuzzReadLong, 0}, step{fuzzReadInt, 0}, step{fuzzReadBytes, 0}, step{fuzzReadIndex, 3}),
			append(AppendLong(AppendLong(nil, 1), math.MaxInt64), v...),
		)
	}

	f.Fuzz(func(t *testing.T, program, data []byte) {
		br := NewBytesReader(data)
		rr := NewBinaryReader(bytes.NewReader(data))
		pos := 0

		for i := 0; i+1 < len(program); i += 2 {
			op := newFuzzRead(program[i], uint16(program[i+1]))
			rest := data[pos:]

			if op.overAllocates(rest) {
				_, err := op.read(br)
				require.Equal(t, io.ErrUnexpectedEOF, requireReaderErr(t, err))
				_, n := op.consume(rest)
				require.Equal(t, io.ErrUnexpectedEOF, ConsumeError(n))
				return
			}

			start := rr.Offset()
			want, wantErr := op.read(rr)
			got, gotErr := op.read(br)
			consumed, n := op.consume(rest)

			requireSameRead(t, want, wantErr, got, gotErr)
			requireSameConsume(t, want, wantErr, rr.Offset()-start, consumed, n)
			if wantErr != nil {
				require.Equal(t, int64(pos), br.Offset(), "a failed bytes-mode read consumed input")
				return
			}

			pos += n
			require.Equal(t, rr.Offset(), br.Offset())
			require.Equal(t, int64(pos), br.Offset())
			require.True(t, bytes.Equal(data[pos:], br.Remaining()))
		}
	})
}

// requireConsumed asserts that a Consume function decoded the whole of
// encoded, which is followed in input by trailing bytes it must leave alone,
// and that n matches the Size function's result.
func requireConsumed(t *testing.T, encoded []byte, n, size int) {
	t.Helper()

	require.NoError(t, ConsumeError(n))
	require.Equal(t, len(encoded), n)
	require.Equal(t, size, n)
}

// fuzzTrailing follows every encoding FuzzAppend_roundTrip consumes, so that a
// Consume function reading past its value is caught.
var fuzzTrailing = []byte{0x02, 0x04}

func withTrailing(encoded []byte) []byte {
	return append(bytes.Clone(encoded), fuzzTrailing...)
}

// FuzzAppend_roundTrip appends fuzzed values and consumes them back. Each must
// decode to the value appended, with n equal to the number of bytes appended
// and to the matching Size function's result. Bool, float, double and fixed
// have no Size function; their fixed sizes stand in for one.
func FuzzAppend_roundTrip(f *testing.F) {
	f.Add(int64(0), int32(0), uint32(0), false, float32(0), float64(0), []byte{}, "")
	f.Add(int64(-1), int32(-1), uint32(1), true, float32(3.14), -2.718281828459045, []byte{0x00, 0xff, 0x7f}, "hello, avro")
	f.Add(int64(math.MaxInt64), int32(math.MaxInt32), uint32(math.MaxUint32), true, float32(math.Inf(1)), math.Inf(-1), bytes.Repeat([]byte{0xab}, 64), "héllo, 世界")
	f.Add(int64(math.MinInt64), int32(math.MinInt32), uint32(63), false, float32(math.NaN()), math.NaN(), bytes.Repeat([]byte{0xab}, 200), "")
	f.Add(int64(64), int32(-65), uint32(64), true, float32(math.SmallestNonzeroFloat32), math.SmallestNonzeroFloat64, []byte(nil), "\x00")

	f.Fuzz(func(t *testing.T, l int64, i int32, idx uint32, bo bool, fl float32, d float64, b []byte, s string) {
		t.Run("long", func(t *testing.T) {
			encoded := AppendLong(nil, l)
			v, n := ConsumeLong(withTrailing(encoded))
			requireConsumed(t, encoded, n, SizeLong(l))
			require.Equal(t, l, v)
		})

		t.Run("int", func(t *testing.T) {
			encoded := AppendInt(nil, i)
			v, n := ConsumeInt(withTrailing(encoded))
			requireConsumed(t, encoded, n, SizeLong(int64(i)))
			require.Equal(t, i, v)
		})

		t.Run("index", func(t *testing.T) {
			index := int(idx)
			encoded := AppendIndex(nil, index)
			v, n := ConsumeIndex(withTrailing(encoded), index+1)
			requireConsumed(t, encoded, n, SizeIndex(index))
			require.Equal(t, index, v)
		})

		t.Run("bool", func(t *testing.T) {
			encoded := AppendBool(nil, bo)
			v, n := ConsumeBool(withTrailing(encoded))
			requireConsumed(t, encoded, n, 1)
			require.Equal(t, bo, v)
		})

		t.Run("float", func(t *testing.T) {
			encoded := AppendFloat(nil, fl)
			v, n := ConsumeFloat(withTrailing(encoded))
			requireConsumed(t, encoded, n, 4)
			require.Equal(t, math.Float32bits(fl), math.Float32bits(v))
		})

		t.Run("double", func(t *testing.T) {
			encoded := AppendDouble(nil, d)
			v, n := ConsumeDouble(withTrailing(encoded))
			requireConsumed(t, encoded, n, 8)
			require.Equal(t, math.Float64bits(d), math.Float64bits(v))
		})

		t.Run("bytes", func(t *testing.T) {
			encoded := AppendBytes(nil, b)
			v, n := ConsumeBytes(withTrailing(encoded))
			requireConsumed(t, encoded, n, SizeBytes(len(b)))
			require.True(t, bytes.Equal(b, v), "decoded %x, appended %x", v, b)
		})

		t.Run("string", func(t *testing.T) {
			encoded := AppendString(nil, s)
			v, n := ConsumeBytes(withTrailing(encoded))
			requireConsumed(t, encoded, n, SizeBytes(len(s)))
			require.Equal(t, s, string(v))
		})

		t.Run("fixed", func(t *testing.T) {
			encoded := AppendFixed(nil, b)
			v, n := ConsumeFixed(withTrailing(encoded), len(b))
			requireConsumed(t, encoded, n, len(b))
			require.True(t, bytes.Equal(b, v), "decoded %x, appended %x", v, b)
		})
	})
}
