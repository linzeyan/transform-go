package convert

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

type StructField struct {
	GoName     string
	JSONName   string
	TypeExpr   ast.Expr
	TypeString string
	Comment    string
	Tag        string
}

type StructDefinition struct {
	Name   string
	Fields []StructField
}

func parseGoStructDefinitions(src string) ([]StructDefinition, error) {
	source := strings.TrimSpace(src)
	if source == "" {
		return nil, errors.New("empty input")
	}
	if !strings.Contains(source, "package ") {
		source = "package main\n" + source
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "input.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var defs []StructDefinition
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			def := buildStructDefinition(ts.Name.Name, st, fileSet)
			defs = append(defs, def)
		}
	}
	if len(defs) == 0 {
		return nil, errors.New("no struct definition found")
	}
	return defs, nil
}

func buildStructDefinition(name string, st *ast.StructType, fileSet *token.FileSet) StructDefinition {
	def := StructDefinition{Name: name}
	for _, field := range st.Fields.List {
		comment := strings.TrimSpace(fieldDoc(field))
		jsonName := ""
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			jsonName = parseJSONTag(tag)
		}
		names := field.Names
		if len(names) == 0 {
			// embedded field
			ident := &ast.Ident{Name: exprString(field.Type, fileSet)}
			names = []*ast.Ident{ident}
		}
		for _, ident := range names {
			if jsonName == "" {
				jsonName = lowerFirst(ident.Name)
			}
			def.Fields = append(def.Fields, StructField{
				GoName:     ident.Name,
				JSONName:   jsonName,
				TypeExpr:   field.Type,
				TypeString: exprString(field.Type, fileSet),
				Comment:    comment,
				Tag:        tagLiteral(field.Tag),
			})
		}
	}
	return def
}

func fieldDoc(field *ast.Field) string {
	if field.Doc != nil {
		return field.Doc.Text()
	}
	if field.Comment != nil {
		return field.Comment.Text()
	}
	return ""
}

func parseJSONTag(tag string) string {
	if tag == "" {
		return ""
	}
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "json:\"") {
			p := strings.TrimPrefix(part, "json:\"")
			p = strings.TrimSuffix(p, "\"")
			if p == "-" {
				return ""
			}
			if idx := strings.Index(p, ","); idx >= 0 {
				return p[:idx]
			}
			return p
		}
	}
	return ""
}

func tagLiteral(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	return tag.Value
}

func exprString(expr ast.Expr, fileSet *token.FileSet) string {
	var buf bytes.Buffer
	_ = format.Node(&buf, fileSet, expr)
	return buf.String()
}
