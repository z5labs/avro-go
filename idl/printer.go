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
	if f.Schema == nil {
		return nil
	}
	return printNamespace
}

func printNamespace(pr *printer, f *File) printerAction {
	if f.Schema.Namespace != "" {
		pr.writef("namespace %s;\n", f.Schema.Namespace)
	}
	return printComments(0)
}

func printComments(idx int) printerAction {
	return func(pr *printer, f *File) printerAction {
		if idx >= len(f.Comments) {
			return printSchemaKeyword
		}
		pr.writef("%s\n", f.Comments[idx].Text)
		return printComments(idx + 1)
	}
}

func printSchemaKeyword(pr *printer, f *File) printerAction {
	if f.Schema.Type == nil {
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
		switch typ := t.(type) {
		case Ident:
			pr.write(typ.Value)
		}
		return next
	}
}
