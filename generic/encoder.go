// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"io"

	"github.com/z5labs/avro-go"
)

// Encoder encodes generic Values against a compiled avro.Schema.
//
// An Encoder is read-only after construction and is safe for concurrent use as
// long as callers supply their own *avro.BinaryWriter per call.
type Encoder struct {
	enc encodeFn
}

// NewEncoder validates s and returns an Encoder that encodes Values matching
// it. Schema-level errors (unresolved Ref, duplicate field names, invalid
// logical-type pairing, etc.) are reported here.
func NewEncoder(s avro.Schema) (*Encoder, error) {
	if s == nil {
		return nil, ErrNilSchema
	}
	node, err := compileSchema(s, newCompileCtx())
	if err != nil {
		return nil, err
	}
	return &Encoder{enc: node.enc}, nil
}

// Encode writes v to w using the compiled schema.
func (e *Encoder) Encode(w io.Writer, v Value) error {
	return e.enc(avro.NewBinaryWriter(w), v)
}
