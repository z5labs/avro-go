// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import "io"

// File represents a parsed Avro IDL file.
type File struct{}

// Parse the Avro IDL defined in the given reader.
func Parse(r io.Reader) (*File, error) {
	return &File{}, nil
}
