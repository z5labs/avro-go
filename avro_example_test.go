// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

type message struct {
	content string
}

func (m message) MarshalAvroBinary(w *BinaryWriter) error {
	return w.WriteString(m.content)
}

func ExampleMarshalBinary() {
	var buf bytes.Buffer

	err := MarshalBinary(&buf, message{content: "abc"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(hex.Dump(buf.Bytes()))

	// Output: 00000000  06 61 62 63                                       |.abc|
}
