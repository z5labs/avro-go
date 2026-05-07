// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package generic

import (
	"fmt"
	"strings"

	"github.com/z5labs/avro-go"
)

// fullName returns the fully-qualified Avro name for a named type.
func fullName(name, namespace string) string {
	if name == "" {
		return ""
	}
	if strings.Contains(name, ".") || namespace == "" {
		return name
	}
	return namespace + "." + name
}

// namedKind tracks the kind of a registered named schema, used to validate
// that Ref-references resolve to compatible types.
type namedKind int

const (
	namedRecord namedKind = iota + 1
	namedEnum
	namedFixed
)

// namedEntry holds a compiled named-type entry. The encode/decode pointers are
// late-bound so that recursive types can resolve their own references.
type namedEntry struct {
	kind   namedKind
	schema avro.Schema
	enc    *encodeFn
	dec    *decodeFn
	size   int // for Fixed
	// For Enum: pre-built symbol → index lookup.
	symbolIndex map[string]int
	symbols     []string
}

// encodeFn / decodeFn are pointer-indirected so Ref nodes can be wired to a
// late-resolved closure.
type (
	encodeFn func(w *avro.BinaryWriter, v Value) error
	decodeFn func(r *avro.BinaryReader) (Value, error)
)

// compileCtx is shared state across schema compilation.
type compileCtx struct {
	named map[string]*namedEntry
}

func newCompileCtx() *compileCtx {
	return &compileCtx{named: map[string]*namedEntry{}}
}

// resolveNamed returns the named entry for the fully-qualified name, or an
// error if it has not been registered. Caller must use a pointer-deref pattern
// when wiring encoders/decoders for Ref nodes so that recursive registrations
// work.
func (c *compileCtx) resolveNamed(name, namespace string) (*namedEntry, error) {
	full := fullName(name, namespace)
	entry, ok := c.named[full]
	if !ok {
		return nil, fmt.Errorf("avro/generic: unresolved reference %q", full)
	}
	return entry, nil
}

// registerNamed registers a named type. Returns an error if the name is
// already registered.
func (c *compileCtx) registerNamed(name, namespace string, entry *namedEntry) error {
	full := fullName(name, namespace)
	if full == "" {
		return fmt.Errorf("avro/generic: named type missing name")
	}
	if _, exists := c.named[full]; exists {
		return fmt.Errorf("avro/generic: duplicate named type %q", full)
	}
	c.named[full] = entry
	return nil
}
