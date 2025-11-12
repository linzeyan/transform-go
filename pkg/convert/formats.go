package convert

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func JSONToYAML(input string) (string, error) {
	data, err := decodeJSONValue(input)
	if err != nil {
		return "", err
	}
	return encodeYAML(data)
}

func YAMLToJSON(input string) (string, error) {
	var data interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return "", err
	}
	normalized := normalizeYAML(data)
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(normalized); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func JSONToTOML(input string) (string, error) {
	data, err := decodeJSONValue(input)
	if err != nil {
		return "", err
	}
	obj, ok := data.(map[string]any)
	if !ok {
		return "", errors.New("TOML root must be an object")
	}
	out, err := toml.Marshal(normalizeJSONNumbers(obj).(map[string]any))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func TOMLToJSON(input string) (string, error) {
	var data map[string]any
	if err := toml.Unmarshal([]byte(input), &data); err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func JSONToSchema(input string) (string, error) {
	data, err := decodeJSONValue(input)
	if err != nil {
		return "", err
	}
	schema := buildSchema(data)
	formatted, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func SchemaToJSON(input string) (string, error) {
	schema, err := decodeJSONValue(input)
	if err != nil {
		return "", err
	}
	sample := sampleFromSchema(schema)
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sample); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func SchemaToGoStruct(input string) (string, error) {
	jsonStr, err := SchemaToJSON(input)
	if err != nil {
		return "", err
	}
	return JSONToGoStruct(jsonStr)
}

func YAMLToGoStruct(input string) (string, error) {
	jsonStr, err := YAMLToJSON(input)
	if err != nil {
		return "", err
	}
	return JSONToGoStruct(jsonStr)
}

func TOMLToGoStruct(input string) (string, error) {
	jsonStr, err := TOMLToJSON(input)
	if err != nil {
		return "", err
	}
	return JSONToGoStruct(jsonStr)
}

func GoStructToYAML(src string) (string, error) {
	jsonStr, err := GoStructToJSON(src)
	if err != nil {
		return "", err
	}
	return JSONToYAML(jsonStr)
}

func GoStructToTOML(src string) (string, error) {
	jsonStr, err := GoStructToJSON(src)
	if err != nil {
		return "", err
	}
	return JSONToTOML(jsonStr)
}

func GoStructToSchema(src string) (string, error) {
	jsonStr, err := GoStructToJSON(src)
	if err != nil {
		return "", err
	}
	return JSONToSchema(jsonStr)
}

func decodeJSONValue(input string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func buildSchema(v any) map[string]any {
	switch val := v.(type) {
	case map[string]any:
		props := make(map[string]any, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			props[k] = buildSchema(val[k])
		}
		schema := map[string]any{
			"type":       "object",
			"properties": props,
		}
		if len(keys) > 0 {
			schema["required"] = keys
		}
		return schema
	case []any:
		schema := map[string]any{
			"type": "array",
		}
		var sample any
		for _, item := range val {
			if item != nil {
				sample = item
				break
			}
		}
		if sample == nil && len(val) > 0 {
			sample = val[0]
		}
		if sample == nil {
			schema["items"] = map[string]any{"type": "string"}
		} else {
			schema["items"] = buildSchema(sample)
		}
		return schema
	case json.Number:
		return map[string]any{"type": "number"}
	case string:
		return map[string]any{"type": "string"}
	case bool:
		return map[string]any{"type": "boolean"}
	case nil:
		return map[string]any{"type": "null"}
	default:
		return map[string]any{"type": "string"}
	}
}

func sampleFromSchema(schema any) any {
	switch s := schema.(type) {
	case map[string]any:
		switch schemaType(s) {
		case "array":
			items, ok := s["items"]
			if !ok {
				return []any{}
			}
			return []any{sampleFromSchema(items)}
		case "string":
			if def, ok := s["default"]; ok {
				return def
			}
			if enums, ok := s["enum"].([]any); ok && len(enums) > 0 {
				return enums[0]
			}
			return ""
		case "number", "integer":
			if def, ok := s["default"]; ok {
				return def
			}
			return 0.0
		case "boolean":
			if def, ok := s["default"]; ok {
				return def
			}
			return false
		case "null":
			return nil
		case "object":
			props := map[string]any{}
			if m, ok := s["properties"].(map[string]any); ok {
				keys := make([]string, 0, len(m))
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					props[k] = sampleFromSchema(m[k])
				}
			}
			return props
		default:
			return map[string]any{}
		}
	case []any:
		if len(s) == 0 {
			return nil
		}
		return sampleFromSchema(s[0])
	default:
		return nil
	}
}

func schemaType(m map[string]any) string {
	switch t := m["type"].(type) {
	case string:
		return t
	case []any:
		for _, v := range t {
			if s, ok := v.(string); ok && s != "null" {
				return s
			}
		}
		if len(t) > 0 {
			if s, ok := t[0].(string); ok {
				return s
			}
		}
	}
	return "object"
}
