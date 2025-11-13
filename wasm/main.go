//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/linzeyan/transform-go/pkg/convert"
)

func main() {
	registerBindings(js.Global())
	select {}
}

type converter func(string) (string, error)

func registerBindings(target js.Value) {
	bindings := map[string]converter{
		"goStructToGraphQL": convert.GoStructToGraphQL,
		"goStructToJSON":    convert.GoStructToJSON,
		"goStructToProto":   convert.GoStructToProto,
		"goStructToSchema":  convert.GoStructToSchema,
		"goStructToTOML":    convert.GoStructToTOML,
		"goStructToYAML":    convert.GoStructToYAML,

		"graphQLToJSON": convert.GraphQLToJSON,

		"jsonToGoStruct": convert.JSONToGoStruct,
		"jsonToGraphQL":  convert.JSONToGraphQL,
		"jsonToProto":    convert.JSONToProto,
		"jsonToSchema":   convert.JSONToSchema,
		"jsonToTOML":     convert.JSONToTOML,
		"jsonToYAML":     convert.JSONToYAML,

		"protobufToJSON": convert.ProtoToJSON,

		"schemaToGoStruct": convert.SchemaToGoStruct,
		"schemaToJSON":     convert.SchemaToJSON,

		"tomlToGoStruct": convert.TOMLToGoStruct,
		"tomlToJSON":     convert.TOMLToJSON,

		"yamlToGoStruct": convert.YAMLToGoStruct,
		"yamlToJSON":     convert.YAMLToJSON,
	}
	for name, fn := range bindings {
		bind(target, name, fn)
	}

	target.Set("transformFormat", js.FuncOf(transformFormat))
	target.Set("formatContent", js.FuncOf(formatContent))
}

var boundHandlers []js.Func

func bind(target js.Value, name string, fn converter) {
	handler := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return map[string]any{"error": "missing input"}
		}
		out, err := fn(args[0].String())
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"result": out}
	})
	boundHandlers = append(boundHandlers, handler)
	target.Set(name, handler)
}

func transformFormat(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "from, to, input required"}
	}
	from := args[0].String()
	to := args[1].String()
	input := args[2].String()
	out, err := convert.ConvertFormats(from, to, input)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}

func formatContent(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "format, input, minify required"}
	}
	formatName := args[0].String()
	input := args[1].String()
	minify := args[2].Bool()
	out, err := convert.FormatContent(formatName, input, minify)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}
