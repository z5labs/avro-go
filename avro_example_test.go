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

func (m *message) UnmarshalAvroBinary(r *BinaryReader) error {
	var err error
	m.content, err = r.ReadString()
	return err
}

func ExampleMarshalBinary() {
	var buf bytes.Buffer

	err := MarshalBinary(&buf, message{content: "abc"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(hex.Dump(buf.Bytes()))

	// Output: 00000000  06 61 62 63                                       |.abc|
}

func ExampleUnmarshalBinary() {
	data := []byte{0x06, 0x61, 0x62, 0x63}

	var msg message
	err := UnmarshalBinary(bytes.NewReader(data), &msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(msg.content)

	// Output: abc
}
