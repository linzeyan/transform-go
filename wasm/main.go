//go:build js && wasm

package main

import (
	"fmt"
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

type formatAdapter struct {
	toJSON   func(string) (string, error)
	fromJSON func(string) (string, error)
}

var formatAdapters = map[string]formatAdapter{
	"JSON": {
		toJSON:   func(s string) (string, error) { return s, nil },
		fromJSON: func(s string) (string, error) { return s, nil },
	},
	"Go Struct": {
		toJSON:   convert.GoStructToJSON,
		fromJSON: convert.JSONToGoStruct,
	},
	"YAML": {
		toJSON:   convert.YAMLToJSON,
		fromJSON: convert.JSONToYAML,
	},
	"TOML": {
		toJSON:   convert.TOMLToJSON,
		fromJSON: convert.JSONToTOML,
	},
	"JSON Schema": {
		toJSON:   convert.SchemaToJSON,
		fromJSON: convert.JSONToSchema,
	},
	"GraphQL Schema": {
		toJSON:   convert.GraphQLToJSON,
		fromJSON: convert.JSONToGraphQL,
	},
	"Protobuf": {
		toJSON:   convert.ProtoToJSON,
		fromJSON: convert.JSONToProto,
	},
}

func transformFormat(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "from, to, input required"}
	}
	from := args[0].String()
	to := args[1].String()
	input := args[2].String()
	out, err := convertFormats(from, to, input)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}

func convertFormats(from, to, input string) (string, error) {
	switch {
	case from == to:
		return input, nil
	case from == "Go Struct" && to == "GraphQL Schema":
		return convert.GoStructToGraphQL(input)
	case from == "GraphQL Schema" && to == "Go Struct":
		return convert.GraphQLToGoStruct(input)
	case from == "Go Struct" && to == "Protobuf":
		return convert.GoStructToProto(input)
	case from == "Protobuf" && to == "Go Struct":
		return convert.ProtoToGoStruct(input)
	}
	fromAdapter, ok := formatAdapters[from]
	if !ok {
		return "", fmt.Errorf("unsupported source format: %s", from)
	}
	toAdapter, ok := formatAdapters[to]
	if !ok {
		return "", fmt.Errorf("unsupported target format: %s", to)
	}
	var mid string
	var err error
	if from == "JSON" {
		mid = input
	} else if fromAdapter.toJSON != nil {
		mid, err = fromAdapter.toJSON(input)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("format %s cannot convert to JSON", from)
	}
	if to == "JSON" {
		return mid, nil
	}
	if toAdapter.fromJSON == nil {
		return "", fmt.Errorf("format %s cannot be generated from JSON", to)
	}
	return toAdapter.fromJSON(mid)
}
