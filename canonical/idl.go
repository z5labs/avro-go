// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package canonical

import (
	"fmt"
	"strings"

	"github.com/z5labs/avro-go/idl"
)

var primitives = map[string]Primitive{
	"null":    Null,
	"boolean": Boolean,
	"int":     Int,
	"long":    Long,
	"float":   Float,
	"double":  Double,
	"bytes":   Bytes,
	"string":  String,
}

// SchemaFrom converts an [idl.Schema] to a slice of canonical [Schema] values.
// Each named type in the IDL schema produces one canonical Schema.
// If the IDL schema has a single Type, a single-element slice is returned.
func SchemaFrom(schema *idl.Schema) ([]Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("canonical: nil schema")
	}

	if schema.Type != nil {
		s, err := schemaFromType(schema.Namespace, schema.Type)
		if err != nil {
			return nil, err
		}
		return []Schema{s}, nil
	}

	schemas := make([]Schema, 0, len(schema.Types))
	for _, t := range schema.Types {
		s, err := schemaFromType(schema.Namespace, t)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, s)
	}
	return schemas, nil
}

func schemaFromType(namespace string, t idl.Type) (Schema, error) {
	switch v := t.(type) {
	case idl.Ident:
		return schemaFromIdent(namespace, v), nil
	case idl.Record:
		return schemaFromRecord(namespace, v)
	case idl.Enum:
		return schemaFromEnum(namespace, v), nil
	case idl.Array:
		return schemaFromArray(namespace, v)
	case idl.Map:
		return schemaFromMap(namespace, v), nil
	case idl.Union:
		return schemaFromUnion(namespace, v)
	case idl.Fixed:
		return schemaFromFixed(namespace, v), nil
	default:
		return Schema{}, fmt.Errorf("canonical: unknown idl type %T", t)
	}
}

func schemaFromIdent(namespace string, ident idl.Ident) Schema {
	if p, ok := primitives[ident.Value]; ok {
		return PrimitiveSchema(p)
	}
	return PrimitiveSchema(Primitive(qualifyName(namespace, ident.Value)))
}

func schemaFromRecord(namespace string, r idl.Record) (Schema, error) {
	ns := effectiveNamespace(namespace, r.Namespace)
	fields := make([]Field, len(r.Fields))
	for i, f := range r.Fields {
		ft, err := schemaFromType(ns, f.Type)
		if err != nil {
			return Schema{}, err
		}
		fields[i] = Field{
			Name: f.Name,
			Type: ft,
		}
	}
	return RecordSchema(Record{
		Name:   qualifyName(ns, r.Name),
		Fields: fields,
	}), nil
}

func schemaFromEnum(namespace string, e idl.Enum) Schema {
	ns := effectiveNamespace(namespace, e.Namespace)
	symbols := make([]string, len(e.Values))
	for i, v := range e.Values {
		symbols[i] = v.Value
	}
	return EnumSchema(Enum{
		Name:    qualifyName(ns, e.Name),
		Symbols: symbols,
	})
}

func schemaFromArray(namespace string, a idl.Array) (Schema, error) {
	items, err := schemaFromType(namespace, a.Items)
	if err != nil {
		return Schema{}, err
	}
	return ArraySchema(Array{Items: items}), nil
}

func schemaFromMap(namespace string, m idl.Map) Schema {
	return MapSchema(Map{Values: schemaFromIdent(namespace, *m.Values)})
}

func schemaFromUnion(namespace string, u idl.Union) (Schema, error) {
	schemas := make(Union, len(u.Types))
	for i, t := range u.Types {
		s, err := schemaFromType(namespace, t)
		if err != nil {
			return Schema{}, err
		}
		schemas[i] = s
	}
	return UnionSchema(schemas), nil
}

func schemaFromFixed(namespace string, f idl.Fixed) Schema {
	ns := effectiveNamespace(namespace, f.Namespace)
	return FixedSchema(Fixed{
		Name: qualifyName(ns, f.Name),
		Size: f.Size,
	})
}

func qualifyName(namespace, name string) string {
	if strings.Contains(name, ".") || namespace == "" {
		return name
	}
	return namespace + "." + name
}

func effectiveNamespace(schemaNamespace, typeNamespace string) string {
	if typeNamespace != "" {
		return typeNamespace
	}
	return schemaNamespace
}
