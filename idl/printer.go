// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"fmt"
	"io"
	"slices"
)

// Print the given File to the given writer in Avro IDL format.
func Print(w io.Writer, f *File) error {
	pr := &printer{w: w}
	for action := printFile; action != nil && pr.err == nil; {
		action = action(pr, f)
	}
	return pr.err
}

type printer struct {
	w   io.Writer
	err error
}

func (pr *printer) write(s string) {
	if pr.err != nil {
		return
	}
	_, pr.err = io.WriteString(pr.w, s)
}

func (pr *printer) writef(format string, args ...any) {
	if pr.err != nil {
		return
	}
	_, pr.err = fmt.Fprintf(pr.w, format, args...)
}

type printerAction func(pr *printer, f *File) printerAction

// writeThen writes a string and returns the next action.
func writeThen(s string, next printerAction) printerAction {
	return func(pr *printer, f *File) printerAction {
		pr.write(s)
		return next
	}
}

func printFile(pr *printer, f *File) printerAction {
	if f == nil {
		pr.err = fmt.Errorf("idl: cannot print nil File")
		return nil
	}
	if f.Schema == nil {
		pr.err = fmt.Errorf("idl: unsupported or empty top-level construct: Schema is nil")
		return nil
	}
	return printFileContent(0)
}

// printFileContent handles interleaving namespace, comments, and schema based on position.
// The logic: namespace (if present) comes first, then comments before schema, then schema.
// For comments before namespace, they are printed first only if they appear before the namespace
// would logically be (i.e., before the Schema.Pos line when namespace is empty, or we have
// no way to track original namespace position, so we assume namespace comes first if present).
func printFileContent(commentIdx int) printerAction {
	return func(pr *printer, f *File) printerAction {
		// If namespace exists, print it first, then handle comments
		if f.Schema.Namespace != "" {
			pr.writef("namespace %s;\n", f.Schema.Namespace)
			return printCommentsBeforeSchema(commentIdx)
		}

		// No namespace: print comments that come before the schema
		if commentIdx < len(f.Comments) {
			comment := f.Comments[commentIdx]
			if comment.Pos.Line < f.Schema.Pos.Line {
				pr.writef("%s\n", comment.Text)
				return printFileContent(commentIdx + 1)
			}
		}

		return printCommentsBeforeSchema(commentIdx)
	}
}

// printCommentsBeforeSchema prints remaining comments then the schema keyword.
func printCommentsBeforeSchema(commentIdx int) printerAction {
	return func(pr *printer, f *File) printerAction {
		if commentIdx < len(f.Comments) {
			pr.writef("%s\n", f.Comments[commentIdx].Text)
			return printCommentsBeforeSchema(commentIdx + 1)
		}
		return printSchemaKeyword
	}
}

func printSchemaKeyword(pr *printer, f *File) printerAction {
	if f.Schema.Type == nil {
		pr.err = fmt.Errorf("idl: Schema.Type is nil")
		return nil
	}
	pr.write("schema ")
	return printSchemaType
}

func printSchemaType(pr *printer, f *File) printerAction {
	return printType(f.Schema.Type, writeThen(";", printSchemaTypes(0)))
}

// printSchemaTypes prints the type definitions in Schema.Types.
func printSchemaTypes(idx int) printerAction {
	return func(pr *printer, f *File) printerAction {
		if idx >= len(f.Schema.Types) {
			return nil
		}
		if idx == 0 {
			pr.write("\n")
		}
		return printType(f.Schema.Types[idx], printSchemaTypes(idx+1))
	}
}

func printType(t Type, next printerAction) printerAction {
	return func(pr *printer, f *File) printerAction {
		if pr.err != nil {
			return nil
		}

		switch typ := t.(type) {
		case Ident:
			pr.write(typ.Value)
		case *Enum:
			return printEnum(typ, next)
		case *Union:
			return printUnion(typ, next)
		default:
			pr.err = fmt.Errorf("idl: unsupported schema type %T in printer", typ)
			return nil
		}
		return next
	}
}

// printUnion prints a union type. Uses shorthand syntax (type?) for nullable
// unions (exactly null + one other type), verbose syntax otherwise.
func printUnion(u *Union, next printerAction) printerAction {
	if isNullableUnion(u) {
		return printType(u.Types[1], writeThen("?", next))
	}
	return printUnionVerbose(u, next)
}

// isNullableUnion returns true if union is exactly [null, T].
func isNullableUnion(u *Union) bool {
	if len(u.Types) != 2 {
		return false
	}
	if ident, ok := u.Types[0].(Ident); ok && ident.Value == "null" {
		return true
	}
	return false
}

// printUnionVerbose prints union { type1, type2, ... } format.
func printUnionVerbose(u *Union, next printerAction) printerAction {
	return writeThen("union { ", printUnionTypes(u.Types, 0, writeThen(" }", next)))
}

// printUnionTypes prints union member types with comma separators.
func printUnionTypes(types []Type, idx int, next printerAction) printerAction {
	if idx >= len(types) {
		return next
	}
	if idx > 0 {
		return writeThen(", ", printType(types[idx], printUnionTypes(types, idx+1, next)))
	}
	return printType(types[idx], printUnionTypes(types, idx+1, next))
}

// printEnum prints an enum type definition.
func printEnum(e *Enum, next printerAction) printerAction {
	return printDoc(e.Doc,
		printNamespaceAnnotation(e.Namespace,
			printAliasesAnnotation(e.Aliases,
				printProperties(e.Properties,
					printEnumKeywordAndName(e.Name,
						printEnumValues(e.Values, 0,
							printEnumDefault(e.Default, next)))))))
}

// printDoc prints a doc comment if non-empty.
func printDoc(doc string, next printerAction) printerAction {
	if doc == "" {
		return next
	}
	return func(pr *printer, f *File) printerAction {
		pr.writef("/** %s */\n", doc)
		return next
	}
}

// printNamespaceAnnotation prints a @namespace annotation if non-empty.
func printNamespaceAnnotation(ns string, next printerAction) printerAction {
	if ns == "" {
		return next
	}
	return func(pr *printer, f *File) printerAction {
		pr.writef("@namespace(\"%s\")\n", ns)
		return next
	}
}

// printAliasesAnnotation prints an @aliases annotation if non-empty.
func printAliasesAnnotation(aliases []string, next printerAction) printerAction {
	if len(aliases) == 0 {
		return next
	}
	return func(pr *printer, f *File) printerAction {
		pr.write("@aliases([")
		for i, alias := range aliases {
			if i > 0 {
				pr.write(", ")
			}
			pr.writef("\"%s\"", alias)
		}
		pr.write("])\n")
		return next
	}
}

// printProperties prints custom @property annotations in sorted key order.
func printProperties(props map[string]Value, next printerAction) printerAction {
	if len(props) == 0 {
		return next
	}
	return func(pr *printer, f *File) printerAction {
		keys := make([]string, 0, len(props))
		for name := range props {
			keys = append(keys, name)
		}
		slices.Sort(keys)
		for _, name := range keys {
			pr.writef("@%s(", name)
			printValue(pr, props[name])
			pr.write(")\n")
		}
		return next
	}
}

// printValue prints a Value (used in annotations and defaults).
func printValue(pr *printer, v Value) {
	if v == nil {
		pr.err = fmt.Errorf("idl: cannot print nil Value")
		return
	}
	switch val := v.(type) {
	case NullValue:
		pr.write("null")
	case BoolValue:
		if val {
			pr.write("true")
		} else {
			pr.write("false")
		}
	case IntValue:
		pr.writef("%d", int64(val))
	case FloatValue:
		pr.writef("%g", float64(val))
	case StringValue:
		pr.writef("\"%s\"", string(val))
	case ArrayValue:
		pr.write("[")
		for i, elem := range val {
			if i > 0 {
				pr.write(", ")
			}
			printValue(pr, elem)
		}
		pr.write("]")
	case ObjectValue:
		pr.write("{")
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for i, k := range keys {
			if i > 0 {
				pr.write(", ")
			}
			pr.writef("\"%s\": ", k)
			printValue(pr, val[k])
		}
		pr.write("}")
	default:
		pr.err = fmt.Errorf("idl: unsupported Value type %T in printer", v)
	}
}

// printEnumKeywordAndName prints "enum Name ".
func printEnumKeywordAndName(name string, next printerAction) printerAction {
	return func(pr *printer, f *File) printerAction {
		pr.writef("enum %s ", name)
		return next
	}
}

// printEnumValues prints the enum values, each on its own line.
func printEnumValues(values []*Ident, idx int, next printerAction) printerAction {
	return func(pr *printer, f *File) printerAction {
		if idx == 0 {
			pr.write("{\n")
		}
		if idx >= len(values) {
			pr.write("}")
			return next
		}
		pr.writef("  %s", values[idx].Value)
		if idx < len(values)-1 {
			pr.write(",")
		}
		pr.write("\n")
		return printEnumValues(values, idx+1, next)
	}
}

// printEnumDefault prints the enum default value if present, then a newline.
func printEnumDefault(def *Ident, next printerAction) printerAction {
	return func(pr *printer, f *File) printerAction {
		if def != nil {
			pr.writef(" = %s;", def.Value)
		}
		pr.write("\n")
		return next
	}
}
