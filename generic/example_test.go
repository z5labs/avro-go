// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/z5labs/avro-go"
)

// exampleWriter adapts an *Encoder/Value pair to avro.BinaryMarshaler so it
// can be passed to avro.MarshalBinary.
type exampleWriter struct {
	enc *Encoder
	v   Value
}

func (s exampleWriter) MarshalAvroBinary(w *avro.BinaryWriter) error {
	return s.enc.Encode(w, s.v)
}

// exampleReader adapts a *Decoder to avro.BinaryUnmarshaler.
type exampleReader struct {
	dec *Decoder
	out *Value
}

func (s exampleReader) UnmarshalAvroBinary(r *avro.BinaryReader) error {
	v, err := s.dec.Decode(r)
	if err != nil {
		return err
	}
	*s.out = v
	return nil
}

// ExampleEncoder shows the canonical "encode a record" flow against a schema.
func ExampleEncoder() {
	schema := avro.Record{
		Name: "User",
		Fields: []*avro.Field{
			{Name: "id", Type: avro.Long{}},
			{Name: "name", Type: avro.String{}},
		},
	}

	enc, err := NewEncoder(schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	user := Record{Fields: []Field{
		{Name: "id", Value: Long(1)},
		{Name: "name", Value: String("ada")},
	}}

	var buf bytes.Buffer
	if err := avro.MarshalBinary(&buf, exampleWriter{enc: enc, v: user}); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(hex.Dump(buf.Bytes()))
	// Output:
	// 00000000  02 06 61 64 61                                    |..ada|
}

// ExampleDecoder shows the canonical "decode a union" flow.
func ExampleDecoder() {
	schema := avro.Union{Types: []avro.Schema{avro.Null{}, avro.String{}}}
	dec, err := NewDecoder(schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Wire bytes for union index 1 + string "hi".
	data := []byte{0x02, 0x04, 'h', 'i'}

	var v Value
	if err := avro.UnmarshalBinary(bytes.NewReader(data), exampleReader{dec: dec, out: &v}); err != nil {
		fmt.Println(err)
		return
	}

	u := v.(Union)
	fmt.Printf("index=%d value=%q\n", u.Index, string(u.Value.(String)))
	// Output: index=1 value="hi"
}

// ExampleEncoder_reuse shows that one Encoder can be reused across many
// records — the schema-validation cost is paid once.
func ExampleEncoder_reuse() {
	schema := avro.Record{
		Name: "Point",
		Fields: []*avro.Field{
			{Name: "x", Type: avro.Long{}},
			{Name: "y", Type: avro.Long{}},
		},
	}
	enc, err := NewEncoder(schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	points := []Record{
		{Fields: []Field{{Name: "x", Value: Long(0)}, {Name: "y", Value: Long(0)}}},
		{Fields: []Field{{Name: "x", Value: Long(1)}, {Name: "y", Value: Long(2)}}},
		{Fields: []Field{{Name: "x", Value: Long(3)}, {Name: "y", Value: Long(4)}}},
	}

	var buf bytes.Buffer
	for _, p := range points {
		if err := avro.MarshalBinary(&buf, exampleWriter{enc: enc, v: p}); err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(buf.Len(), "bytes,", len(points), "records, single Encoder")
	// Output: 6 bytes, 3 records, single Encoder
}
