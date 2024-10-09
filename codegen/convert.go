package codegen

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"text/template"
)

//go:embed convert.tpl
var convertTPL string

// RPCConversion creates functions for converting between core newsdoc strucs
// and their protobuf Message equivalents.
func RPCConversion(
	w io.Writer,
	protoPackage string, source string, formatter string,
) error {
	tmpl, err := template.New("convert").Parse(convertTPL)
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
		Messages []Message
	}{
		Package:  protoPackage,
		Messages: messages,
	}

	var rawGo bytes.Buffer

	err = tmpl.Execute(&rawGo, data)
	if err != nil {
		return fmt.Errorf("failed to render teplate: %w", err)
	}

	formatterPath, err := exec.LookPath(formatter)
	if err != nil {
		return fmt.Errorf("failed to locate formatter: %w", err)
	}

	cmd := exec.Cmd{
		Path:   formatterPath,
		Stdin:  &rawGo,
		Stdout: w,
		Stderr: os.Stderr,
	}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run %q: %w", formatter, err)
	}

	return nil
}
