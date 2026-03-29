// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"errors"
	"io"
)

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

var (
	// ErrBadMagic is returned when the single-object header bytes are not [0xC3, 0x01].
	ErrBadMagic = errors.New("avro: bad single-object magic bytes")

	// ErrFingerprintMismatch is returned when the fingerprint in the payload
	// does not match the expected fingerprint from the unmarshaler.
	ErrFingerprintMismatch = errors.New("avro: fingerprint mismatch")
)

// SingleObjectUnmarshaler is the interface implemented by types that support
// Avro single-object decoding. It embeds BinaryUnmarshaler and adds a method
// to return the expected 8-byte CRC-64-AVRO fingerprint of the schema.
type SingleObjectUnmarshaler interface {
	BinaryUnmarshaler

	// Fingerprint returns the 8-byte little-endian CRC-64-AVRO fingerprint
	// of the object's Avro schema.
	Fingerprint() [8]byte
}

// UnmarshalSingleObject reads an Avro single-object encoded payload from r.
// It validates the 2-byte magic header and 8-byte schema fingerprint, then
// delegates binary decoding to the unmarshaler.
func UnmarshalSingleObject(r io.Reader, obj SingleObjectUnmarshaler) error {
	var header [2]byte
	_, err := io.ReadFull(r, header[:])
	if err != nil {
		return err
	}
	if header[0] != 0xC3 || header[1] != 0x01 {
		return ErrBadMagic
	}

	var fp [8]byte
	_, err = io.ReadFull(r, fp[:])
	if err != nil {
		return err
	}
	if fp != obj.Fingerprint() {
		return ErrFingerprintMismatch
	}

	return obj.UnmarshalAvroBinary(&BinaryReader{in: r})
}
