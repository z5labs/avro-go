// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical_test

import (
	"encoding/json"
	"fmt"

	"github.com/z5labs/avro-go/canonical"
	"github.com/z5labs/avro-go/idl"
)

func ExampleSchemaFrom() {
	idlSchema := &idl.Schema{
		Namespace: "com.example",
		Type: idl.Record{
			Name: "Person",
			Fields: []*idl.Field{
				{Name: "name", Type: idl.Ident{Value: "string"}},
				{Name: "age", Type: idl.Ident{Value: "int"}},
			},
		},
	}

	schemas, err := canonical.SchemaFrom(idlSchema)
	if err != nil {
		fmt.Println(err)
		return
	}

	b, err := json.Marshal(schemas[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"name":"com.example.Person","type":"record","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
}
