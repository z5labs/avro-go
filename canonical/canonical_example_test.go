// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical_test

import (
	"encoding/json"
	"fmt"

	"github.com/z5labs/avro-go/canonical"
)

func ExamplePrimitiveSchema() {
	s := canonical.PrimitiveSchema(canonical.String)

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: "string"
}

func ExampleRecordSchema() {
	s := canonical.RecordSchema(canonical.Record{
		Name: "com.example.Person",
		Fields: []canonical.Field{
			{Name: "name", Type: canonical.PrimitiveSchema(canonical.String)},
			{Name: "age", Type: canonical.PrimitiveSchema(canonical.Int)},
		},
	})

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"name":"com.example.Person","type":"record","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
}

func ExampleEnumSchema() {
	s := canonical.EnumSchema(canonical.Enum{
		Name:    "com.example.Color",
		Symbols: []string{"RED", "GREEN", "BLUE"},
	})

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"name":"com.example.Color","type":"enum","symbols":["RED","GREEN","BLUE"]}
}

func ExampleArraySchema() {
	s := canonical.ArraySchema(canonical.Array{
		Items: canonical.PrimitiveSchema(canonical.String),
	})

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"type":"array","items":"string"}
}

func ExampleMapSchema() {
	s := canonical.MapSchema(canonical.Map{
		Values: canonical.PrimitiveSchema(canonical.Int),
	})

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"type":"map","values":"int"}
}

func ExampleUnionSchema() {
	s := canonical.UnionSchema(canonical.Union{
		canonical.PrimitiveSchema(canonical.Null),
		canonical.PrimitiveSchema(canonical.String),
	})

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: ["null","string"]
}

func ExampleFixedSchema() {
	s := canonical.FixedSchema(canonical.Fixed{
		Name: "com.example.MD5",
		Size: 16,
	})

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"name":"com.example.MD5","type":"fixed","size":16}
}

func ExampleSchema_UnmarshalJSON() {
	data := []byte(`{"name":"com.example.Person","type":"record","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}`)

	var s canonical.Schema
	err := json.Unmarshal(data, &s)
	if err != nil {
		fmt.Println(err)
		return
	}

	r, ok := s.Record()
	if !ok {
		fmt.Println("not a record")
		return
	}

	fmt.Println(r.Name)
	for _, f := range r.Fields {
		p, _ := f.Type.Primitive()
		fmt.Printf("  %s: %s\n", f.Name, p)
	}

	// Output:
	// com.example.Person
	//   name: string
	//   age: int
}
