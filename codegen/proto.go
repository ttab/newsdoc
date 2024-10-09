package codegen

import (
	_ "embed"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"text/template"
)

//go:embed proto.tpl
var protoTPL string

// Protobuf generates protobuf declarations from Go structs.
func Protobuf(
	w io.Writer,
	protoPackage string, source string, options map[string]string,
) error {
	tmpl, err := template.New("proto").Parse(protoTPL)
	if err != nil {
		return fmt.Errorf("parse protobuf template: %w", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, source, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse Go source: %w", err)
	}

	messages, err := InterpretAST(fset, file)
	if err != nil {
		return fmt.Errorf("interpreting Go AST: %w", err)
	}

	data := struct {
		Package  string
		Options  map[string]string
		Messages []Message
	}{
		Package:  protoPackage,
		Options:  options,
		Messages: messages,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return fmt.Errorf("failed to render teplate: %w", err)
	}

	return nil
}
