// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

// Package generic provides schema-driven encoding and decoding of arbitrary
// Avro values, similar to Java's GenericRecord/GenericData or protobuf's
// dynamicpb.
//
// Values are constructed from the Value types in this package (Null, Bool,
// Int, Long, Float, Double, Bytes, String, Record, Enum, Array, Map, Union,
// Fixed, plus the twelve logical-type values) and do not carry their own
// schema. The schema is supplied to NewEncoder or NewDecoder, which validate
// it once and pre-compute lookup tables that every Encode/Decode call reuses.
//
// An Encoder or Decoder is read-only after construction and is safe for
// concurrent use across goroutines as long as each call supplies its own
// *avro.BinaryWriter or *avro.BinaryReader.
package generic
