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
		"jsonToGoStruct":   convert.JSONToGoStruct,
		"goStructToJSON":   convert.GoStructToJSON,
		"goStructToYAML":   convert.GoStructToYAML,
		"goStructToTOML":   convert.GoStructToTOML,
		"goStructToSchema": convert.GoStructToSchema,
		"jsonToYAML":       convert.JSONToYAML,
		"yamlToJSON":       convert.YAMLToJSON,
		"jsonToTOML":       convert.JSONToTOML,
		"tomlToJSON":       convert.TOMLToJSON,
		"yamlToGoStruct":   convert.YAMLToGoStruct,
		"tomlToGoStruct":   convert.TOMLToGoStruct,
		"jsonToSchema":     convert.JSONToSchema,
		"schemaToJSON":     convert.SchemaToJSON,
		"schemaToGoStruct": convert.SchemaToGoStruct,
	}
	for name, fn := range bindings {
		bind(target, name, fn)
	}
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
