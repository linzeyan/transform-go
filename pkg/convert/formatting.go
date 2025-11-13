package convert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"strings"
)

const (
	formatJSON     = "JSON"
	formatGoStruct = "Go Struct"
	formatYAML     = "YAML"
	formatTOML     = "TOML"
	formatSchema   = "JSON Schema"
	formatGraphQL  = "GraphQL Schema"
	formatProtobuf = "Protobuf"
)

type formatAdapter struct {
	ToJSON   func(string) (string, error)
	FromJSON func(string) (string, error)
}

var adapters = map[string]formatAdapter{
	formatJSON: {
		ToJSON:   func(s string) (string, error) { return s, nil },
		FromJSON: func(s string) (string, error) { return s, nil },
	},
	formatGoStruct: {
		ToJSON:   GoStructToJSON,
		FromJSON: JSONToGoStruct,
	},
	formatYAML: {
		ToJSON:   YAMLToJSON,
		FromJSON: JSONToYAML,
	},
	formatTOML: {
		ToJSON:   TOMLToJSON,
		FromJSON: JSONToTOML,
	},
	formatSchema: {
		ToJSON:   SchemaToJSON,
		FromJSON: JSONToSchema,
	},
	formatGraphQL: {
		ToJSON:   GraphQLToJSON,
		FromJSON: JSONToGraphQL,
	},
	formatProtobuf: {
		ToJSON:   ProtoToJSON,
		FromJSON: JSONToProto,
	},
}

func ConvertFormats(from, to, input string) (string, error) {
	switch {
	case from == to:
		return input, nil
	case from == formatGoStruct && to == formatGraphQL:
		return GoStructToGraphQL(input)
	case from == formatGraphQL && to == formatGoStruct:
		return GraphQLToGoStruct(input)
	case from == formatGoStruct && to == formatProtobuf:
		return GoStructToProto(input)
	case from == formatProtobuf && to == formatGoStruct:
		return ProtoToGoStruct(input)
	}
	fromAdapter, ok := adapters[from]
	if !ok {
		return "", fmt.Errorf("unsupported source format: %s", from)
	}
	toAdapter, ok := adapters[to]
	if !ok {
		return "", fmt.Errorf("unsupported target format: %s", to)
	}
	var mid string
	var err error
	if from == formatJSON {
		mid = input
	} else if fromAdapter.ToJSON != nil {
		mid, err = fromAdapter.ToJSON(input)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("format %s cannot convert to JSON", from)
	}
	if to == formatJSON {
		return mid, nil
	}
	if toAdapter.FromJSON == nil {
		return "", fmt.Errorf("format %s cannot be generated from JSON", to)
	}
	return toAdapter.FromJSON(mid)
}

func FormatContent(formatName, input string, minify bool) (string, error) {
	switch formatName {
	case formatGoStruct:
		return formatGoSource(input)
	case formatJSON:
		return normalizeJSONOutput(input, minify)
	}
	adapter, ok := adapters[formatName]
	if !ok {
		return "", fmt.Errorf("unsupported format: %s", formatName)
	}
	if adapter.ToJSON == nil || adapter.FromJSON == nil {
		return "", fmt.Errorf("format %s cannot be formatted", formatName)
	}
	jsonStr, err := adapter.ToJSON(input)
	if err != nil {
		return "", err
	}
	normalized, err := normalizeJSONOutput(jsonStr, minify)
	if err != nil {
		return "", err
	}
	return adapter.FromJSON(normalized)
}

func normalizeJSONOutput(input string, minify bool) (string, error) {
	if minify {
		var buf bytes.Buffer
		if err := json.Compact(&buf, []byte(input)); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	var v any
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func formatGoSource(src string) (string, error) {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return "", nil
	}
	prefixed := trimmed
	hasPackage := strings.Contains(trimmed, "package ")
	if !hasPackage {
		prefixed = "package main\n\n" + trimmed
	}
	formatted, err := format.Source([]byte(prefixed))
	if err != nil {
		return "", err
	}
	out := string(formatted)
	if !hasPackage {
		if idx := strings.Index(out, "\n\n"); idx >= 0 {
			out = strings.TrimSpace(out[idx+2:])
		}
	}
	return out, nil
}
