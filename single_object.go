// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import "io"

// SingleObjectMarshaler is the interface implemented by types that support
// Avro single-object encoding. It embeds BinaryMarshaler and adds a method
// to return the precomputed 8-byte CRC-64-AVRO fingerprint of the schema.
type SingleObjectMarshaler interface {
	BinaryMarshaler

	// Fingerprint returns the 8-byte little-endian CRC-64-AVRO fingerprint
	// of the object's Avro schema.
	Fingerprint() [8]byte
}

// MarshalSingleObject writes the Avro single-object encoding to w.
// The format is: 2-byte magic header (0xC3 0x01), 8-byte schema fingerprint,
// followed by the Avro binary encoded data.
func MarshalSingleObject(w io.Writer, obj SingleObjectMarshaler) error {
	n, err := w.Write([]byte{0xC3, 0x01})
	if err != nil {
		return err
	}
	if n != 2 {
		return io.ErrShortWrite
	}

	fp := obj.Fingerprint()
	n, err = w.Write(fp[:])
	if err != nil {
		return err
	}
	if n != 8 {
		return io.ErrShortWrite
	}

	return obj.MarshalAvroBinary(&BinaryWriter{out: w})
}
