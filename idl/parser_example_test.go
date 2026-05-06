// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl_test

import (
	"fmt"
	"strings"

	"github.com/z5labs/avro-go/idl"
)

func ExampleParse_errors() {
	_, err := idl.Parse(strings.NewReader(`schema ;`))
	if err != nil {
		fmt.Println(err)
	}

	_, err = idl.Parse(strings.NewReader(`schema int `))
	if err != nil {
		fmt.Println(err)
	}

	// Output:
	// unexpected token at line 1, column 8: Symbol(;), expected one of: Identifier
	// unexpected end of tokens at line 1, column 8, expected one of: Symbol
}
