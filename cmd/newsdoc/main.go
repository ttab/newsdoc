package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	jsv "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/ttab/newsdoc"
	"github.com/ttab/newsdoc/codegen"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "newsdoc",
		Usage: "NewsDoc tools",
	}

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
			&cli.StringSliceFlag{
				Name:    "option",
				Aliases: []string{"o"},
				Usage:   `-o[ption] go_package=./repository`,
			},
		},
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "jsonschema",
		Action: jsonschemaAction,
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
		options      = c.StringSlice("option")
	)

	protoOpt := make(map[string]string)

	for _, v := range options {
		key, value, ok := strings.Cut(v, "=")
		if !ok {
			return fmt.Errorf("invalid option %q", v)
		}

		protoOpt[key] = value
	}

	err := codegen.Protobuf(os.Stdout, protoPackage, sourceName, protoOpt)
	if err != nil {
		return fmt.Errorf("generate protobuf: %w", err)
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

		pair := def.Properties.Oldest()
		for pair != nil {
			pair.Value.Description = strings.ReplaceAll(
				pair.Value.Description, "\n", " ",
			)

			pair = pair.Next()
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
