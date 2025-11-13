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
	target.Set("encodeContent", js.FuncOf(encodeContent))
	target.Set("decodeContent", js.FuncOf(decodeContent))
	target.Set("hashContent", js.FuncOf(hashContent))
	target.Set("urlEncode", js.FuncOf(urlEncode))
	target.Set("urlDecode", js.FuncOf(urlDecode))
	target.Set("jwtEncode", js.FuncOf(jwtEncode))
	target.Set("jwtDecode", js.FuncOf(jwtDecode))
	target.Set("markdownToHTML", js.FuncOf(markdownToHTML))
	target.Set("htmlToMarkdown", js.FuncOf(htmlToMarkdown))
	target.Set("convertNumberBase", js.FuncOf(convertNumberBase))
	target.Set("ipv4Info", js.FuncOf(ipv4Info))
	target.Set("generateUUIDs", js.FuncOf(generateUUIDs))
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

func encodeContent(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "missing input"}
	}
	out, err := convert.EncodeContent(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": stringMapToAny(out)}
}

func decodeContent(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return map[string]any{"error": "encoding and input required"}
	}
	out, err := convert.DecodeContent(args[0].String(), args[1].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}

func hashContent(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "missing input"}
	}
	out := convert.HashContent(args[0].String())
	return map[string]any{"result": stringMapToAny(out)}
}

func urlEncode(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "missing input"}
	}
	return map[string]any{"result": convert.URLEncode(args[0].String())}
}

func urlDecode(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "missing input"}
	}
	out, err := convert.URLDecode(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}

func jwtEncode(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "payload, secret, algorithm required"}
	}
	token, err := convert.JWTEncode(args[0].String(), args[1].String(), args[2].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": map[string]any{"token": token}}
}

func jwtDecode(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "token required"}
	}
	parts, err := convert.JWTDecode(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": map[string]any{
		"header":    parts.Header,
		"payload":   parts.Payload,
		"signature": parts.Signature,
		"algorithm": parts.Algorithm,
	}}
}

func markdownToHTML(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "missing input"}
	}
	out, err := convert.MarkdownToHTML(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}

func htmlToMarkdown(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "missing input"}
	}
	out, err := convert.HTMLToMarkdown(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": out}
}

func convertNumberBase(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return map[string]any{"error": "base and value required"}
	}
	out, err := convert.ConvertNumberBase(args[0].String(), args[1].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": map[string]any{
		"binary":  out.Binary,
		"octal":   out.Octal,
		"decimal": out.Decimal,
		"hex":     out.Hex,
	}}
}

func ipv4Info(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return map[string]any{"error": "input required"}
	}
	info, err := convert.IPv4Info(args[0].String())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": map[string]any{
		"type":       info.Type,
		"input":      info.Input,
		"cidr":       info.CIDR,
		"mask":       info.Mask,
		"rangeStart": info.RangeStart,
		"rangeEnd":   info.RangeEnd,
		"total":      info.Total,
		"standard":   info.Standard,
		"threePart":  info.ThreePart,
		"twoPart":    info.TwoPart,
		"integer":    info.Integer,
	}}
}

func generateUUIDs(_ js.Value, _ []js.Value) any {
	result, err := convert.GenerateUUIDs()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"result": stringMapToAny(result)}
}

func stringMapToAny(in map[string]string) map[string]any {
	result := make(map[string]any, len(in))
	for k, v := range in {
		result[k] = v
	}
	return result
}
