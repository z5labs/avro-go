// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package idl

import (
	"fmt"
	"io"
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
	return printType(f.Schema.Type, writeThen(";", nil))
}

func printType(t Type, next printerAction) printerAction {
	return func(pr *printer, f *File) printerAction {
		if pr.err != nil {
			return nil
		}

		switch typ := t.(type) {
		case Ident:
			pr.write(typ.Value)
		default:
			pr.err = fmt.Errorf("idl: unsupported schema type %T in printer", typ)
			return nil
		}
		return next
	}
}
