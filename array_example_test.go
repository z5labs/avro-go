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

// reading is an Avro record of a single long, standing in for whatever item
// type a schema puts inside an array.
type reading int64

func (r *reading) MarshalAvroBinary(w *BinaryWriter) error {
	return w.WriteLong(int64(*r))
}

func (r *reading) UnmarshalAvroBinary(br *BinaryReader) error {
	v, err := br.ReadLong()
	if err != nil {
		return err
	}
	*r = reading(v)
	return nil
}

func ExampleArrayWriter() {
	var buf bytes.Buffer

	// Unsized blocks are the default: each item goes straight to the underlying
	// writer as its own block, so nothing is buffered.
	w := NewArrayWriter(NewBinaryWriter(&buf))
	for _, v := range []int64{1, 2, 3} {
		item := reading(v)
		if err := w.Write(&item); err != nil {
			fmt.Println(err)
			return
		}
	}

	// Close writes the terminating zero-count block. Without it the array is
	// truncated, not merely unflushed.
	if err := w.Close(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(hex.Dump(buf.Bytes()))

	// Output: 00000000  02 02 02 04 02 06 00                              |.......|
}

func ExampleWriteArray() {
	var buf bytes.Buffer

	// WriteArray closes the array for you once the callback returns. Sized
	// blocks buffer up to 64 bytes at a time and record each block's encoded
	// size, which is what lets a reader skip whole blocks.
	err := WriteArray(NewBinaryWriter(&buf), func(w *ArrayWriter) error {
		for _, v := range []int64{1, 2, 3} {
			item := reading(v)
			if err := w.Write(&item); err != nil {
				return err
			}
		}
		return nil
	}, WithSizedBlocks(64))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(hex.Dump(buf.Bytes()))

	// Output: 00000000  05 06 02 04 06 00                                 |......|
}

func ExampleArrayReader() {
	// Three items in a single unsized block, then the terminator.
	data := []byte{0x06, 0x02, 0x04, 0x06, 0x00}

	r := NewArrayReader(NewBinaryReader(bytes.NewReader(data)))

	// One destination reused for every item, so memory use is a function of a
	// single item rather than of the array's length.
	var item reading
	for {
		ok, err := r.Next(&item)
		if err != nil {
			fmt.Println(err)
			return
		}
		if !ok {
			break
		}
		fmt.Println(item)
	}

	// Output:
	// 1
	// 2
	// 3
}

func ExampleArrayReader_SkipBlock() {
	// A sized block of three items, then an unsized block of one, then the
	// terminator.
	data := []byte{0x05, 0x06, 0x02, 0x04, 0x06, 0x02, 0x08, 0x00}

	r := NewArrayReader(NewBinaryReader(bytes.NewReader(data)))

	var item reading
	for {
		skipped, err := r.SkipBlock()
		if err != nil {
			fmt.Println(err)
			return
		}
		if skipped == SkipNone {
			break
		}
		if skipped == SkipSized {
			fmt.Println("discarded a sized block without decoding it")
			continue
		}

		// An unsized block carries no byte count, so its end can only be found
		// by decoding. Drain it instead.
		fmt.Println("draining an unsized block")
		for {
			ok, err := r.Next(&item)
			if err != nil {
				fmt.Println(err)
				return
			}
			if !ok {
				return
			}
			fmt.Println(item)
		}
	}

	// Output:
	// discarded a sized block without decoding it
	// draining an unsized block
	// 4
}
