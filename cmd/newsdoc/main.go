package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/invopop/jsonschema"
	jsv "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/ttab/newsdoc"
	"github.com/urfave/cli/v2"
)

//go:embed proto.tpl
var protoTPL string

func main() {
	app := &cli.App{
		Name:  "newsdoc",
		Usage: "NewsDoc tools",
	}

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "jsonschema",
		Action: jsonschemaAction,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "protobuf",
		Action: protobufAction,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "package",
				Value: "newsdoc",
			},
			&cli.PathFlag{
				Name:  "source",
				Value: "doc.go",
			},
		},
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "validate",
		Action: validateAction,
	})

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func protobufAction(c *cli.Context) error {
	var (
		protoPackage = c.String("package")
		sourceName   = c.String("source")
	)

	tmpl, err := template.New("proto").Parse(protoTPL)
	if err != nil {
		return fmt.Errorf("parse protobuf template: %w", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, sourceName, nil, parser.ParseComments)
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

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		return fmt.Errorf("failed to render teplate: %w", err)
	}

	return nil
}

func jsonschemaAction(_ *cli.Context) error {
	var r jsonschema.Reflector

	err := r.AddGoComments("github.com/ttab/newsdoc", "./")
	if err != nil {
		return fmt.Errorf("the schema can only be regenerated in the root of the github.com/ttab/newsdoc package: %w", err)
	}

	s := r.Reflect(&newsdoc.Document{})

	// Remove the comment newlines from the generated descriptions.
	for _, def := range s.Definitions {
		def.Description = strings.ReplaceAll(def.Description, "\n", " ")

		if def.Properties == nil {
			continue
		}

		for _, name := range def.Properties.Keys() {
			prop, _ := def.Properties.Get(name)
			propSchema := prop.(*jsonschema.Schema)

			propSchema.Description = strings.ReplaceAll(
				propSchema.Description, "\n", " ",
			)
		}
	}

	enc := json.NewEncoder(os.Stdout)

	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	err = enc.Encode(s)
	if err != nil {
		return fmt.Errorf("failed to encode jsonschema: %w", err)
	}

	return nil
}

func validateAction(c *cli.Context) error {
	const schemaURL = "https://github.com/ttab/newsdoc/document"

	compiler := jsv.NewCompiler()

	compiler.Draft = jsv.Draft2020
	compiler.AssertFormat = true
	compiler.AssertContent = true

	err := compiler.AddResource(
		schemaURL,
		bytes.NewReader(newsdoc.JSONSchema()),
	)
	if err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var (
		in io.Reader
		v  interface{}
	)

	if c.NArg() == 0 {
		in = os.Stdin
	} else {
		f, err := os.Open(c.Args().First())
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}

		defer f.Close()

		in = f
	}

	dec := json.NewDecoder(in)

	err = dec.Decode(&v)
	if err != nil {
		return fmt.Errorf("failed to decode file: %w", err)
	}

	err = schema.Validate(v)
	if err != nil {
		return fmt.Errorf("%#v", err) //nolint: errorlint
	}

	return nil
}
