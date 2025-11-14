package convert

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/linzeyan/transform-go/pkg/common"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func JSONToYAML(input string) (string, error) {
	data, err := decodeJSONValue(input)
	if err != nil {
		return "", err
	}
	return common.EncodeYAML(common.NormalizeJSONNumbers(data))
}

func YAMLToJSON(input string) (string, error) {
	var data interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return "", err
	}
	normalized := common.NormalizeYAML(data)
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
	out, err := toml.Marshal(common.NormalizeJSONNumbers(obj).(map[string]any))
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

func JSONToXML(input string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return "", err
	}
	builder := &strings.Builder{}
	builder.WriteString(xml.Header)
	buildXML(builder, "root", common.NormalizeJSONNumbers(data), 0)
	return builder.String(), nil
}

func XMLToJSON(input string) (string, error) {
	root, err := parseXML(input)
	if err != nil {
		return "", err
	}
	value := elementToValue(root)
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
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

func buildXML(builder *strings.Builder, name string, value any, indent int) {
	indentation := strings.Repeat("  ", indent)
	switch val := value.(type) {
	case map[string]any:
		builder.WriteString(fmt.Sprintf("%s<%s>\n", indentation, name))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buildXML(builder, k, val[k], indent+1)
		}
		builder.WriteString(fmt.Sprintf("%s</%s>\n", indentation, name))
	case []any:
		for _, item := range val {
			buildXML(builder, name, item, indent)
		}
	default:
		text := fmt.Sprint(val)
		builder.WriteString(fmt.Sprintf("%s<%s>%s</%s>\n", indentation, name, xmlEscape(text), name))
	}
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

type xmlElement struct {
	Name     string
	Value    string
	Children []*xmlElement
}

func parseXML(src string) (*xmlElement, error) {
	decoder := xml.NewDecoder(strings.NewReader(src))
	var stack []*xmlElement
	var root *xmlElement
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			node := &xmlElement{Name: t.Name.Local}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			text := strings.TrimSpace(string(t))
			if text != "" {
				stack[len(stack)-1].Value += text
			}
		case xml.EndElement:
			if len(stack) == 0 {
				continue
			}
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
		}
	}
	if root == nil {
		return nil, errors.New("invalid XML input")
	}
	return root, nil
}

func elementToValue(el *xmlElement) any {
	if len(el.Children) == 0 {
		return el.Value
	}
	result := map[string]any{}
	for _, child := range el.Children {
		val := elementToValue(child)
		if existing, ok := result[child.Name]; ok {
			switch arr := existing.(type) {
			case []any:
				result[child.Name] = append(arr, val)
			default:
				result[child.Name] = []any{arr, val}
			}
		} else {
			result[child.Name] = val
		}
	}
	return result
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
