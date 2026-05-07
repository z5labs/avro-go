// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"fmt"
	"io"

	"github.com/z5labs/avro-go"
)

// Decoder decodes Avro binary data into generic Values using a compiled
// avro.Schema.
//
// A Decoder is read-only after construction and is safe for concurrent use as
// long as callers supply their own *avro.BinaryReader per call.
type Decoder struct {
	dec decodeFn
}

// NewDecoder validates s and returns a Decoder. Schema-level errors are
// reported here, identical to NewEncoder.
func NewDecoder(s avro.Schema) (*Decoder, error) {
	if s == nil {
		return nil, fmt.Errorf("avro/generic: nil schema")
	}
	node, err := compileSchema(s, newCompileCtx())
	if err != nil {
		return nil, err
	}
	return &Decoder{dec: node.dec}, nil
}

// Decode reads the next Value from r using the compiled schema.
func (d *Decoder) Decode(r io.Reader) (Value, error) {
	return d.dec(avro.NewBinaryReader(r))
}
