// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/z5labs/avro-go"
)

// planNode pairs an encode and decode function compiled from one schema node.
type planNode struct {
	enc encodeFn
	dec decodeFn
}

// compileSchema validates s and produces matching encode/decode functions.
// Validation and plan construction happen in a single walk so that NewEncoder
// and NewDecoder report identical schema-level errors.
//
// namespace is the enclosing Avro namespace inherited from a parent named type.
// Unqualified Refs and inline named types use it when their own Namespace is
// empty, matching Avro's name resolution rules.
func compileSchema(s avro.Schema, ctx *compileCtx, namespace string) (*planNode, error) {
	switch t := s.(type) {
	case avro.Null:
		return nullPlan, nil
	case avro.Boolean:
		return booleanPlan, nil
	case avro.Int:
		return intPlan, nil
	case avro.Long:
		return longPlan, nil
	case avro.Float:
		return floatPlan, nil
	case avro.Double:
		return doublePlan, nil
	case avro.Bytes:
		return bytesPlan, nil
	case avro.String:
		return stringPlan, nil
	case avro.Ref:
		return compileRef(t, ctx, namespace)
	case avro.Record:
		return compileRecord(t, ctx, namespace)
	case avro.Enum:
		return compileEnum(t, ctx, namespace)
	case avro.Array:
		return compileArray(t, ctx, namespace)
	case avro.Map:
		return compileMap(t, ctx, namespace)
	case avro.Union:
		return compileUnion(t, ctx, namespace)
	case avro.Fixed:
		return compileFixed(t, ctx, namespace)
	case avro.Decimal:
		return compileDecimal(t, ctx, namespace)
	case avro.UUID:
		return uuidPlan, nil
	case avro.Date:
		return datePlan, nil
	case avro.TimeMillis:
		return timeMillisPlan, nil
	case avro.TimeMicros:
		return timeMicrosPlan, nil
	case avro.TimestampMillis:
		return timestampMillisPlan, nil
	case avro.TimestampMicros:
		return timestampMicrosPlan, nil
	case avro.TimestampNanos:
		return timestampNanosPlan, nil
	case avro.LocalTimestampMillis:
		return localTimestampMillisPlan, nil
	case avro.LocalTimestampMicros:
		return localTimestampMicrosPlan, nil
	case avro.LocalTimestampNanos:
		return localTimestampNanosPlan, nil
	case avro.Duration:
		return compileDuration(t, ctx, namespace)
	default:
		return nil, UnsupportedSchemaError{Schema: s}
	}
}

// ---- primitive plans ----

var (
	nullPlan = &planNode{
		enc: func(_ *avro.BinaryWriter, v Value) error {
			if _, ok := v.(Null); !ok {
				return TypeMismatchError{Expected: "null", Got: v}
			}
			return nil
		},
		dec: func(_ *avro.BinaryReader) (Value, error) { return Null{}, nil },
	}
	booleanPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			b, ok := v.(Bool)
			if !ok {
				return typeMismatch("boolean", v)
			}
			return w.WriteBool(bool(b))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			b, err := r.ReadBool()
			if err != nil {
				return nil, err
			}
			return Bool(b), nil
		},
	}
	intPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			i, ok := v.(Int)
			if !ok {
				return typeMismatch("int", v)
			}
			return w.WriteInt(int32(i))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			i, err := r.ReadInt()
			if err != nil {
				return nil, err
			}
			return Int(i), nil
		},
	}
	longPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			l, ok := v.(Long)
			if !ok {
				return typeMismatch("long", v)
			}
			return w.WriteLong(int64(l))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			l, err := r.ReadLong()
			if err != nil {
				return nil, err
			}
			return Long(l), nil
		},
	}
	floatPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			f, ok := v.(Float)
			if !ok {
				return typeMismatch("float", v)
			}
			return w.WriteFloat(float32(f))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			f, err := r.ReadFloat()
			if err != nil {
				return nil, err
			}
			return Float(f), nil
		},
	}
	doublePlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			d, ok := v.(Double)
			if !ok {
				return typeMismatch("double", v)
			}
			return w.WriteDouble(float64(d))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			d, err := r.ReadDouble()
			if err != nil {
				return nil, err
			}
			return Double(d), nil
		},
	}
	bytesPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			b, ok := v.(Bytes)
			if !ok {
				return typeMismatch("bytes", v)
			}
			return w.WriteBytes([]byte(b))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			b, err := r.ReadBytes()
			if err != nil {
				return nil, err
			}
			return Bytes(b), nil
		},
	}
	stringPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			s, ok := v.(String)
			if !ok {
				return typeMismatch("string", v)
			}
			return w.WriteString(string(s))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			s, err := r.ReadString()
			if err != nil {
				return nil, err
			}
			return String(s), nil
		},
	}
)

// ---- composite plans ----

func compileRef(r avro.Ref, ctx *compileCtx, namespace string) (*planNode, error) {
	ns := r.Namespace
	if ns == "" {
		ns = namespace
	}
	entry, err := ctx.resolveNamed(r.Name, ns)
	if err != nil {
		return nil, err
	}
	return &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error { return (*entry.enc)(w, v) },
		dec: func(rd *avro.BinaryReader) (Value, error) { return (*entry.dec)(rd) },
	}, nil
}

func compileRecord(r avro.Record, ctx *compileCtx, namespace string) (*planNode, error) {
	if r.Name == "" {
		return nil, MissingNameError{Kind: "record"}
	}
	ns := r.Namespace
	if ns == "" {
		ns = namespace
	}

	encPtr := new(encodeFn)
	decPtr := new(decodeFn)
	if err := ctx.registerNamed(r.Name, ns, &namedEntry{
		kind: namedRecord, schema: r, enc: encPtr, dec: decPtr,
	}); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(r.Fields))
	fieldNames := make([]string, len(r.Fields))
	fieldNodes := make([]*planNode, len(r.Fields))
	for i, f := range r.Fields {
		if f == nil {
			return nil, InvalidFieldError{Record: r.Name, Index: i, Reason: "nil field"}
		}
		if f.Name == "" {
			return nil, InvalidFieldError{Record: r.Name, Index: i, Reason: "missing name"}
		}
		if _, dup := seen[f.Name]; dup {
			return nil, DuplicateFieldError{Record: r.Name, Field: f.Name}
		}
		seen[f.Name] = struct{}{}
		if f.Type == nil {
			return nil, InvalidFieldError{Record: r.Name, Index: i, Field: f.Name, Reason: "nil type"}
		}
		fieldNames[i] = f.Name
		node, err := compileSchema(f.Type, ctx, ns)
		if err != nil {
			return nil, FieldCompileError{Record: r.Name, Field: f.Name, Err: err}
		}
		fieldNodes[i] = node
	}

	name := r.Name
	*encPtr = func(w *avro.BinaryWriter, v Value) error {
		rec, ok := v.(Record)
		if !ok {
			return TypeMismatchError{Expected: fmt.Sprintf("record %q", name), Got: v}
		}
		if len(rec.Fields) != len(fieldNames) {
			return FieldCountError{Record: name, Expected: len(fieldNames), Got: len(rec.Fields)}
		}
		for i, f := range rec.Fields {
			if f.Name != fieldNames[i] {
				return FieldNameError{Record: name, Index: i, Expected: fieldNames[i], Got: f.Name}
			}
			if err := fieldNodes[i].enc(w, f.Value); err != nil {
				return err
			}
		}
		return nil
	}
	*decPtr = func(rd *avro.BinaryReader) (Value, error) {
		fields := make([]Field, len(fieldNames))
		for i, fname := range fieldNames {
			val, err := fieldNodes[i].dec(rd)
			if err != nil {
				return nil, err
			}
			fields[i] = Field{Name: fname, Value: val}
		}
		return Record{Fields: fields}, nil
	}
	return &planNode{enc: *encPtr, dec: *decPtr}, nil
}

func compileEnum(e avro.Enum, ctx *compileCtx, namespace string) (*planNode, error) {
	if e.Name == "" {
		return nil, MissingNameError{Kind: "enum"}
	}
	if len(e.Symbols) == 0 {
		return nil, EmptySymbolsError{Enum: e.Name}
	}
	idx := make(map[string]int, len(e.Symbols))
	for i, s := range e.Symbols {
		if _, dup := idx[s]; dup {
			return nil, DuplicateSymbolError{Enum: e.Name, Symbol: s}
		}
		idx[s] = i
	}
	if e.Default != "" {
		if _, ok := idx[e.Default]; !ok {
			return nil, InvalidEnumDefaultError{Enum: e.Name, Default: e.Default}
		}
	}
	ns := e.Namespace
	if ns == "" {
		ns = namespace
	}

	encPtr := new(encodeFn)
	decPtr := new(decodeFn)
	symbols := append([]string(nil), e.Symbols...)
	if err := ctx.registerNamed(e.Name, ns, &namedEntry{
		kind: namedEnum, schema: e, enc: encPtr, dec: decPtr,
		symbolIndex: idx, symbols: symbols,
	}); err != nil {
		return nil, err
	}

	name := e.Name
	*encPtr = func(w *avro.BinaryWriter, v Value) error {
		en, ok := v.(Enum)
		if !ok {
			return TypeMismatchError{Expected: fmt.Sprintf("enum %q", name), Got: v}
		}
		i, ok := idx[en.Symbol]
		if !ok {
			return UnknownSymbolError{Enum: name, Symbol: en.Symbol}
		}
		return w.WriteInt(int32(i))
	}
	*decPtr = func(r *avro.BinaryReader) (Value, error) {
		i, err := r.ReadInt()
		if err != nil {
			return nil, err
		}
		if int(i) < 0 || int(i) >= len(symbols) {
			return nil, IndexOutOfRangeError{Kind: "enum", Name: name, Index: int64(i), Len: len(symbols)}
		}
		return Enum{Symbol: symbols[i]}, nil
	}
	return &planNode{enc: *encPtr, dec: *decPtr}, nil
}

func compileArray(a avro.Array, ctx *compileCtx, namespace string) (*planNode, error) {
	if a.Items == nil {
		return nil, ErrNilArrayItems
	}
	item, err := compileSchema(a.Items, ctx, namespace)
	if err != nil {
		return nil, ItemsCompileError{Err: err}
	}
	return &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			arr, ok := v.(Array)
			if !ok {
				return TypeMismatchError{Expected: "array", Got: v}
			}
			if len(arr) > 0 {
				if err := w.WriteLong(int64(len(arr))); err != nil {
					return err
				}
				for _, it := range arr {
					if err := item.enc(w, it); err != nil {
						return err
					}
				}
			}
			return w.WriteLong(0)
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			out := Array{}
			for {
				count, err := r.ReadLong()
				if err != nil {
					return nil, err
				}
				if count == 0 {
					return out, nil
				}
				if count < 0 {
					// Negative count means the next long is a byte size we can skip past.
					count = -count
					if _, err := r.ReadLong(); err != nil {
						return nil, err
					}
				}
				for i := int64(0); i < count; i++ {
					val, err := item.dec(r)
					if err != nil {
						return nil, err
					}
					out = append(out, val)
				}
			}
		},
	}, nil
}

func compileMap(m avro.Map, ctx *compileCtx, namespace string) (*planNode, error) {
	if m.Values == nil {
		return nil, ErrNilMapValues
	}
	val, err := compileSchema(m.Values, ctx, namespace)
	if err != nil {
		return nil, ValuesCompileError{Err: err}
	}
	return &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			mp, ok := v.(Map)
			if !ok {
				return TypeMismatchError{Expected: "map", Got: v}
			}
			if len(mp) > 0 {
				if err := w.WriteLong(int64(len(mp))); err != nil {
					return err
				}
				for k, vv := range mp {
					if err := w.WriteString(k); err != nil {
						return err
					}
					if err := val.enc(w, vv); err != nil {
						return err
					}
				}
			}
			return w.WriteLong(0)
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			out := Map{}
			for {
				count, err := r.ReadLong()
				if err != nil {
					return nil, err
				}
				if count == 0 {
					return out, nil
				}
				if count < 0 {
					count = -count
					if _, err := r.ReadLong(); err != nil {
						return nil, err
					}
				}
				for i := int64(0); i < count; i++ {
					k, err := r.ReadString()
					if err != nil {
						return nil, err
					}
					vv, err := val.dec(r)
					if err != nil {
						return nil, err
					}
					out[k] = vv
				}
			}
		},
	}, nil
}

func compileUnion(u avro.Union, ctx *compileCtx, namespace string) (*planNode, error) {
	if len(u.Types) == 0 {
		return nil, ErrEmptyUnion
	}
	branches := make([]*planNode, len(u.Types))
	seenUnnamed := make(map[string]struct{})
	seenNamed := make(map[string]struct{})
	for i, t := range u.Types {
		if t == nil {
			return nil, NilBranchError{Index: i}
		}
		if _, nested := t.(avro.Union); nested {
			return nil, NestedUnionError{Index: i}
		}
		key, named := unionBranchKey(t, namespace)
		if named {
			if _, dup := seenNamed[key]; dup {
				return nil, DuplicateBranchError{Key: key, Named: true}
			}
			seenNamed[key] = struct{}{}
		} else {
			if _, dup := seenUnnamed[key]; dup {
				return nil, DuplicateBranchError{Key: key, Named: false}
			}
			seenUnnamed[key] = struct{}{}
		}
		node, err := compileSchema(t, ctx, namespace)
		if err != nil {
			return nil, BranchCompileError{Index: i, Err: err}
		}
		branches[i] = node
	}
	return &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			un, ok := v.(Union)
			if !ok {
				return TypeMismatchError{Expected: "union", Got: v}
			}
			if un.Index < 0 || un.Index >= len(branches) {
				return IndexOutOfRangeError{Kind: "union", Index: int64(un.Index), Len: len(branches)}
			}
			if err := w.WriteLong(int64(un.Index)); err != nil {
				return err
			}
			return branches[un.Index].enc(w, un.Value)
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			i, err := r.ReadLong()
			if err != nil {
				return nil, err
			}
			if i < 0 || i >= int64(len(branches)) {
				return nil, IndexOutOfRangeError{Kind: "union", Index: i, Len: len(branches)}
			}
			val, err := branches[i].dec(r)
			if err != nil {
				return nil, err
			}
			return Union{Index: int(i), Value: val}, nil
		},
	}, nil
}

// unionBranchKey returns the Avro identity key for a union branch and whether
// the key is for a named type. Per the Avro spec, a union may not contain two
// schemas with the same wire type; logical types collapse to the identity of
// their underlying schema so e.g. [int, date] or [bytes, decimal-over-bytes]
// are correctly detected as duplicates.
func unionBranchKey(s avro.Schema, namespace string) (string, bool) {
	resolveNS := func(declared string) string {
		if declared != "" {
			return declared
		}
		return namespace
	}
	switch t := s.(type) {
	case avro.Null:
		return "null", false
	case avro.Boolean:
		return "boolean", false
	case avro.Int:
		return "int", false
	case avro.Long:
		return "long", false
	case avro.Float:
		return "float", false
	case avro.Double:
		return "double", false
	case avro.Bytes:
		return "bytes", false
	case avro.String:
		return "string", false
	case avro.Array:
		return "array", false
	case avro.Map:
		return "map", false
	case avro.Record:
		return fullName(t.Name, resolveNS(t.Namespace)), true
	case avro.Enum:
		return fullName(t.Name, resolveNS(t.Namespace)), true
	case avro.Fixed:
		return fullName(t.Name, resolveNS(t.Namespace)), true
	case avro.Ref:
		return fullName(t.Name, resolveNS(t.Namespace)), true
	case avro.UUID:
		return "string", false
	case avro.Date, avro.TimeMillis:
		return "int", false
	case avro.TimeMicros,
		avro.TimestampMillis, avro.TimestampMicros, avro.TimestampNanos,
		avro.LocalTimestampMillis, avro.LocalTimestampMicros, avro.LocalTimestampNanos:
		return "long", false
	case avro.Decimal:
		switch under := t.Underlying.(type) {
		case avro.Fixed:
			return fullName(under.Name, resolveNS(under.Namespace)), true
		default:
			return "bytes", false
		}
	case avro.Duration:
		name := t.Underlying.Name
		if name == "" {
			name = "duration"
		}
		return fullName(name, resolveNS(t.Underlying.Namespace)), true
	default:
		return fmt.Sprintf("%T", s), false
	}
}

func compileFixed(f avro.Fixed, ctx *compileCtx, namespace string) (*planNode, error) {
	if f.Name == "" {
		return nil, MissingNameError{Kind: "fixed"}
	}
	if f.Size <= 0 {
		return nil, InvalidFixedSizeError{Name: f.Name, Size: f.Size}
	}
	ns := f.Namespace
	if ns == "" {
		ns = namespace
	}
	encPtr := new(encodeFn)
	decPtr := new(decodeFn)
	if err := ctx.registerNamed(f.Name, ns, &namedEntry{
		kind: namedFixed, schema: f, enc: encPtr, dec: decPtr, size: f.Size,
	}); err != nil {
		return nil, err
	}
	name := f.Name
	size := f.Size
	*encPtr = func(w *avro.BinaryWriter, v Value) error {
		fx, ok := v.(Fixed)
		if !ok {
			return TypeMismatchError{Expected: fmt.Sprintf("fixed %q", name), Got: v}
		}
		if len(fx) != size {
			return FixedSizeError{Name: name, Expected: size, Got: len(fx)}
		}
		return w.WriteFixed([]byte(fx))
	}
	*decPtr = func(r *avro.BinaryReader) (Value, error) {
		b, err := r.ReadFixed(size)
		if err != nil {
			return nil, err
		}
		return Fixed(b), nil
	}
	return &planNode{enc: *encPtr, dec: *decPtr}, nil
}

// ---- logical type plans ----

func compileDecimal(d avro.Decimal, ctx *compileCtx, namespace string) (*planNode, error) {
	if d.Precision <= 0 {
		return nil, InvalidPrecisionError{Precision: d.Precision}
	}
	if d.Scale < 0 || d.Scale > d.Precision {
		return nil, InvalidScaleError{Precision: d.Precision, Scale: d.Scale}
	}
	switch under := d.Underlying.(type) {
	case avro.Bytes:
		precision := d.Precision
		limit := precisionLimit(precision)
		return &planNode{
			enc: func(w *avro.BinaryWriter, v Value) error {
				dec, ok := v.(Decimal)
				if !ok {
					return TypeMismatchError{Expected: "decimal", Got: v}
				}
				if dec.Unscaled == nil {
					return ErrNilDecimalUnscaled
				}
				if !decimalFits(dec.Unscaled, limit) {
					return PrecisionOverflowError{Precision: precision}
				}
				return w.WriteBytes(twosComplementBytes(dec.Unscaled))
			},
			dec: func(r *avro.BinaryReader) (Value, error) {
				b, err := r.ReadBytes()
				if err != nil {
					return nil, err
				}
				return Decimal{Unscaled: bytesToBigInt(b)}, nil
			},
		}, nil
	case avro.Fixed:
		if under.Name == "" {
			return nil, MissingNameError{Kind: "fixed"}
		}
		if under.Size <= 0 {
			return nil, InvalidFixedSizeError{Name: under.Name, Size: under.Size}
		}
		size := under.Size
		precision := d.Precision
		limit := precisionLimit(precision)
		ns := under.Namespace
		if ns == "" {
			ns = namespace
		}

		// The underlying fixed is a named type per the Avro spec. Register it
		// so a Ref to its name resolves to the same decimal-over-fixed plan
		// rather than a raw fixed-bytes plan.
		encPtr := new(encodeFn)
		decPtr := new(decodeFn)
		if err := ctx.registerNamed(under.Name, ns, &namedEntry{
			kind: namedFixed, schema: d, enc: encPtr, dec: decPtr, size: size,
		}); err != nil {
			return nil, err
		}
		*encPtr = func(w *avro.BinaryWriter, v Value) error {
			dec, ok := v.(Decimal)
			if !ok {
				return TypeMismatchError{Expected: "decimal", Got: v}
			}
			if dec.Unscaled == nil {
				return ErrNilDecimalUnscaled
			}
			if !decimalFits(dec.Unscaled, limit) {
				return PrecisionOverflowError{Precision: precision}
			}
			b := twosComplementBytes(dec.Unscaled)
			if len(b) > size {
				return DecimalSizeError{Encoded: len(b), FixedSize: size}
			}
			return w.WriteFixed(padTwosComplement(b, size, dec.Unscaled.Sign()))
		}
		*decPtr = func(r *avro.BinaryReader) (Value, error) {
			b, err := r.ReadFixed(size)
			if err != nil {
				return nil, err
			}
			return Decimal{Unscaled: bytesToBigInt(b)}, nil
		}
		return &planNode{enc: *encPtr, dec: *decPtr}, nil
	default:
		return nil, InvalidDecimalUnderlyingError{Underlying: d.Underlying}
	}
}

var (
	uuidPlan = &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			u, ok := v.(UUID)
			if !ok {
				return typeMismatch("uuid", v)
			}
			return w.WriteString(string(u))
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			s, err := r.ReadString()
			if err != nil {
				return nil, err
			}
			return UUID(s), nil
		},
	}
	datePlan = newIntLogical("date",
		func(i int32) Value { return Date(i) },
		func(v Value) (int32, bool) { x, ok := v.(Date); return int32(x), ok },
	)
	timeMillisPlan = newIntLogical("time-millis",
		func(i int32) Value { return TimeMillis(i) },
		func(v Value) (int32, bool) { x, ok := v.(TimeMillis); return int32(x), ok },
	)
	timeMicrosPlan = newLongLogical("time-micros",
		func(l int64) Value { return TimeMicros(l) },
		func(v Value) (int64, bool) { x, ok := v.(TimeMicros); return int64(x), ok },
	)
	timestampMillisPlan = newLongLogical("timestamp-millis",
		func(l int64) Value { return TimestampMillis(l) },
		func(v Value) (int64, bool) { x, ok := v.(TimestampMillis); return int64(x), ok },
	)
	timestampMicrosPlan = newLongLogical("timestamp-micros",
		func(l int64) Value { return TimestampMicros(l) },
		func(v Value) (int64, bool) { x, ok := v.(TimestampMicros); return int64(x), ok },
	)
	timestampNanosPlan = newLongLogical("timestamp-nanos",
		func(l int64) Value { return TimestampNanos(l) },
		func(v Value) (int64, bool) { x, ok := v.(TimestampNanos); return int64(x), ok },
	)
	localTimestampMillisPlan = newLongLogical("local-timestamp-millis",
		func(l int64) Value { return LocalTimestampMillis(l) },
		func(v Value) (int64, bool) { x, ok := v.(LocalTimestampMillis); return int64(x), ok },
	)
	localTimestampMicrosPlan = newLongLogical("local-timestamp-micros",
		func(l int64) Value { return LocalTimestampMicros(l) },
		func(v Value) (int64, bool) { x, ok := v.(LocalTimestampMicros); return int64(x), ok },
	)
	localTimestampNanosPlan = newLongLogical("local-timestamp-nanos",
		func(l int64) Value { return LocalTimestampNanos(l) },
		func(v Value) (int64, bool) { x, ok := v.(LocalTimestampNanos); return int64(x), ok },
	)

)

// compileDuration validates an Avro duration schema and registers its
// underlying Fixed name so Refs can resolve to the same plan, mirroring the
// decimal-over-fixed pattern. An empty Underlying.Name defaults to "duration"
// to match Duration.MarshalJSON's convention.
func compileDuration(d avro.Duration, ctx *compileCtx, namespace string) (*planNode, error) {
	under := d.Underlying
	if under.Size != 0 && under.Size != 12 {
		return nil, InvalidFixedSizeError{Name: under.Name, Size: under.Size}
	}
	name := under.Name
	if name == "" {
		name = "duration"
	}
	ns := under.Namespace
	if ns == "" {
		ns = namespace
	}

	encPtr := new(encodeFn)
	decPtr := new(decodeFn)
	if err := ctx.registerNamed(name, ns, &namedEntry{
		kind: namedFixed, schema: d, enc: encPtr, dec: decPtr, size: 12,
	}); err != nil {
		return nil, err
	}
	*encPtr = func(w *avro.BinaryWriter, v Value) error {
		dv, ok := v.(Duration)
		if !ok {
			return typeMismatch("duration", v)
		}
		var buf [12]byte
		binary.LittleEndian.PutUint32(buf[0:4], dv.Months)
		binary.LittleEndian.PutUint32(buf[4:8], dv.Days)
		binary.LittleEndian.PutUint32(buf[8:12], dv.Millis)
		return w.WriteFixed(buf[:])
	}
	*decPtr = func(r *avro.BinaryReader) (Value, error) {
		b, err := r.ReadFixed(12)
		if err != nil {
			return nil, err
		}
		return Duration{
			Months: binary.LittleEndian.Uint32(b[0:4]),
			Days:   binary.LittleEndian.Uint32(b[4:8]),
			Millis: binary.LittleEndian.Uint32(b[8:12]),
		}, nil
	}
	return &planNode{enc: *encPtr, dec: *decPtr}, nil
}

// newIntLogical builds a plan for an int-backed logical type. unwrap converts
// a Value to its int32 wire form (returning ok=false on type mismatch); wrap
// builds the Value back from a decoded int32.
func newIntLogical(name string, wrap func(int32) Value, unwrap func(Value) (int32, bool)) *planNode {
	return &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			i, ok := unwrap(v)
			if !ok {
				return typeMismatch(name, v)
			}
			return w.WriteInt(i)
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			i, err := r.ReadInt()
			if err != nil {
				return nil, err
			}
			return wrap(i), nil
		},
	}
}

// newLongLogical is the int64 counterpart of newIntLogical.
func newLongLogical(name string, wrap func(int64) Value, unwrap func(Value) (int64, bool)) *planNode {
	return &planNode{
		enc: func(w *avro.BinaryWriter, v Value) error {
			l, ok := unwrap(v)
			if !ok {
				return typeMismatch(name, v)
			}
			return w.WriteLong(l)
		},
		dec: func(r *avro.BinaryReader) (Value, error) {
			l, err := r.ReadLong()
			if err != nil {
				return nil, err
			}
			return wrap(l), nil
		},
	}
}

// ---- helpers ----

func typeMismatch(want string, got Value) error {
	return TypeMismatchError{Expected: want, Got: got}
}

// twosComplementBytes returns the minimum-length two's-complement big-endian
// byte representation of x, matching the Avro decimal wire format.
func twosComplementBytes(x *big.Int) []byte {
	if x.Sign() == 0 {
		return []byte{0}
	}
	if x.Sign() > 0 {
		b := x.Bytes()
		if b[0]&0x80 != 0 {
			return append([]byte{0x00}, b...)
		}
		return b
	}
	mag := new(big.Int).Neg(x)
	bits := mag.BitLen()
	byteLen := (bits + 7) / 8
	if bits%8 == 0 {
		byteLen++
	}
	mod := new(big.Int).Lsh(big.NewInt(1), uint(8*byteLen))
	twos := new(big.Int).Sub(mod, mag)
	out := make([]byte, byteLen)
	tb := twos.Bytes()
	copy(out[byteLen-len(tb):], tb)
	return out
}

// bytesToBigInt is the inverse of twosComplementBytes.
func bytesToBigInt(b []byte) *big.Int {
	if len(b) == 0 {
		return big.NewInt(0)
	}
	x := new(big.Int).SetBytes(b)
	if b[0]&0x80 != 0 {
		// Negative: subtract 2^(8*len).
		mod := new(big.Int).Lsh(big.NewInt(1), uint(8*len(b)))
		x.Sub(x, mod)
	}
	return x
}

func padTwosComplement(b []byte, size, sign int) []byte {
	if len(b) == size {
		return b
	}
	pad := byte(0x00)
	if sign < 0 {
		pad = 0xff
	}
	out := make([]byte, size)
	for i := range out[:size-len(b)] {
		out[i] = pad
	}
	copy(out[size-len(b):], b)
	return out
}

// precisionLimit returns 10^precision as a *big.Int. It is computed once per
// compiled decimal plan so encoding does not re-allocate on every call.
func precisionLimit(precision int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
}

// decimalFits reports whether x's magnitude is strictly less than limit.
func decimalFits(x, limit *big.Int) bool {
	return new(big.Int).Abs(x).Cmp(limit) < 0
}
