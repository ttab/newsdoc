package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
)

type Message struct {
	Name    string
	Comment []string
	Fields  []Field
}

type Field struct {
	Name        string
	GoName      string
	Type        string
	GoType      string
	FieldNumber int
	Comment     []string
}

func InterpretAST(set *token.FileSet, file *ast.File) ([]Message, error) {
	var (
		currentType *Message
		messages    []Message
		comments    []string
		inspectErr  error
		depth       int
	)

	processField := func(field *ast.Field) error {
		aTag := field.Tag
		if aTag == nil {
			return nil
		}

		tag, err := structtag.Parse(strings.Trim(aTag.Value, "`"))
		if err != nil {
			return fmt.Errorf(
				"invalid struct tag %q: %w",
				aTag.Value, err,
			)
		}

		jsonT, err := tag.Get("json")
		if err != nil {
			// The error is used as an ok value, an
			// error means "doesn't exist".
			return nil
		}

		protoT, err := tag.Get("proto")
		if err != nil {
			// The error is used as an ok value, an
			// error means "doesn't exist".
			return nil
		}

		f := Field{
			Name:   jsonT.Name,
			GoName: field.Names[0].String(),
		}

		fieldNum, err := strconv.Atoi(protoT.Name)
		if err != nil {
			return fmt.Errorf(
				"invalid protobuf field number: %w", err)
		}

		f.FieldNumber = fieldNum

		ft, err := printExpr(set, field.Type)
		if err != nil {
			return fmt.Errorf(
				"failed to print Go AST expression for field type: %w", err)
		}

		f.GoType = ft

		switch ft {
		case "string":
			f.Type = "string"
		case "DataMap":
			f.Type = "map<string, string>"
		case "[]Block":
			f.Type = "repeated Block"
		default:
			return fmt.Errorf("unknown field type %q", ft)
		}

		for _, c := range field.Doc.List {
			f.Comment = append(f.Comment, c.Text)
		}

		currentType.Fields = append(currentType.Fields, f)

		return nil
	}

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			depth--

			return true
		}

		depth++

		switch n := node.(type) {
		case *ast.CommentGroup:
			if depth != 3 {
				break
			}

			comments = make([]string, 0, len(n.List))

			for _, c := range n.List {
				comments = append(comments, c.Text)
			}
		case *ast.TypeSpec:
			t := Message{
				Name:    n.Name.String(),
				Comment: comments,
			}

			comments = nil
			currentType = &t
		case *ast.StructType:
			if currentType == nil {
				break
			}

			for _, field := range n.Fields.List {
				err := processField(field)
				if err != nil {
					inspectErr = fmt.Errorf(
						"field %s.%s: %w",
						currentType.Name, field.Names[0].Name, err)

					return false
				}
			}

			messages = append(messages, *currentType)
		}

		return true
	})

	if inspectErr != nil {
		return nil, inspectErr
	}

	return messages, nil
}

func printExpr(set *token.FileSet, e ast.Expr) (string, error) {
	var buf bytes.Buffer

	err := printer.Fprint(&buf, set, e)
	if err != nil {
		return "", fmt.Errorf("fprint: %w", err)
	}

	return buf.String(), nil
}
