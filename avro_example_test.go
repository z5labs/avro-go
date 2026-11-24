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

func ExampleUnmarshalBytes() {
	data := []byte{0x06, 0x61, 0x62, 0x63}

	var msg message
	err := UnmarshalBytes(data, &msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(msg.content)

	// Output: abc
}

func ExampleBinaryReader_ResetBytes() {
	records := [][]byte{
		{0x06, 0x61, 0x62, 0x63},
		{0x04, 0x78, 0x79},
	}

	// One reader, reset before each record, decodes without allocating the
	// reader again for every record.
	r := NewBytesReader(nil)
	for _, data := range records {
		r.ResetBytes(data)

		var msg message
		if err := msg.UnmarshalAvroBinary(r); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(msg.content, len(r.Remaining()))
	}

	// Output:
	// abc 0
	// xy 0
}
