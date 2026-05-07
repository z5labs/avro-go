// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic_test

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/z5labs/avro-go"
	"github.com/z5labs/avro-go/generic"
)

// schemaWriter adapts an *Encoder/Value pair to avro.BinaryMarshaler so it can
// be passed to avro.MarshalBinary.
type schemaWriter struct {
	enc *generic.Encoder
	v   generic.Value
}

func (s schemaWriter) MarshalAvroBinary(w *avro.BinaryWriter) error {
	return s.enc.Encode(w, s.v)
}

// schemaReader adapts a *Decoder to avro.BinaryUnmarshaler.
type schemaReader struct {
	dec *generic.Decoder
	out *generic.Value
}

func (s schemaReader) UnmarshalAvroBinary(r *avro.BinaryReader) error {
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

	enc, err := generic.NewEncoder(schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	user := generic.Record{Fields: []generic.Field{
		{Name: "id", Value: generic.Long(1)},
		{Name: "name", Value: generic.String("ada")},
	}}

	var buf bytes.Buffer
	if err := avro.MarshalBinary(&buf, schemaWriter{enc: enc, v: user}); err != nil {
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
	dec, err := generic.NewDecoder(schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Wire bytes for union index 1 + string "hi".
	data := []byte{0x02, 0x04, 'h', 'i'}

	var v generic.Value
	if err := avro.UnmarshalBinary(bytes.NewReader(data), schemaReader{dec: dec, out: &v}); err != nil {
		fmt.Println(err)
		return
	}

	u := v.(generic.Union)
	fmt.Printf("index=%d value=%q\n", u.Index, string(u.Value.(generic.String)))
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
	enc, err := generic.NewEncoder(schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	points := []generic.Record{
		{Fields: []generic.Field{{Name: "x", Value: generic.Long(0)}, {Name: "y", Value: generic.Long(0)}}},
		{Fields: []generic.Field{{Name: "x", Value: generic.Long(1)}, {Name: "y", Value: generic.Long(2)}}},
		{Fields: []generic.Field{{Name: "x", Value: generic.Long(3)}, {Name: "y", Value: generic.Long(4)}}},
	}

	var buf bytes.Buffer
	for _, p := range points {
		if err := avro.MarshalBinary(&buf, schemaWriter{enc: enc, v: p}); err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(buf.Len(), "bytes,", len(points), "records, single Encoder")
	// Output: 6 bytes, 3 records, single Encoder
}
