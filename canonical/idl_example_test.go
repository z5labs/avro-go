// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/z5labs/avro-go/canonical"
	"github.com/z5labs/avro-go/idl"
)

func ExampleSchemaFrom() {
	f, err := idl.Parse(strings.NewReader(`
		namespace com.example;
		schema int;
		record Person {
			string name;
			int age;
		}
	`))
	if err != nil {
		fmt.Println(err)
		return
	}

	schemas, err := canonical.SchemaFrom(f.Schema)
	if err != nil {
		fmt.Println(err)
		return
	}

	b, err := json.Marshal(schemas[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(b))

	// Output: {"name":"com.example.Person","type":"record","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
}
