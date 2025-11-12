package convert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

func looksInteger(num json.Number) bool {
	if strings.ContainsRune(num.String(), '.') {
		return false
	}
	_, err := num.Int64()
	return err == nil
}

func normalizeJSONNumbers(v any) any {
	switch val := v.(type) {
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, vv := range val {
			res[k] = normalizeJSONNumbers(vv)
		}
		return res
	case []any:
		for i, item := range val {
			val[i] = normalizeJSONNumbers(item)
		}
		return val
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	default:
		return v
	}
}

func normalizeYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, vv := range val {
			res[k] = normalizeYAML(vv)
		}
		return res
	case map[interface{}]interface{}:
		res := make(map[string]any, len(val))
		for k, vv := range val {
			key := fmt.Sprint(k)
			res[key] = normalizeYAML(vv)
		}
		return res
	case []interface{}:
		for i, item := range val {
			val[i] = normalizeYAML(item)
		}
		return val
	default:
		return val
	}
}

func encodeYAML(data any) (string, error) {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		_ = enc.Close()
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func jsonFieldName(field *ast.Field) string {
	if field.Tag != nil {
		tag := strings.Trim(field.Tag.Value, "`")
		val := reflect.StructTag(tag).Get("json")
		if val != "" && val != "-" {
			part := strings.Split(val, ",")[0]
			if part != "" {
				return part
			}
		}
	}
	if len(field.Names) == 0 {
		return ""
	}
	return lowerFirst(field.Names[0].Name)
}

func exportName(key string) string {
	var runes []rune
	capNext := true
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capNext {
				runes = append(runes, unicode.ToUpper(r))
				capNext = false
			} else {
				runes = append(runes, r)
			}
		} else {
			capNext = true
		}
	}
	out := string(runes)
	out = strings.TrimLeftFunc(out, func(r rune) bool {
		return !unicode.IsLetter(r) && r != '_'
	})
	return out
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	words := splitWords(s)
	if len(words) == 0 {
		return strings.ToLower(s)
	}
	var buf strings.Builder
	buf.WriteString(strings.ToLower(words[0]))
	for _, word := range words[1:] {
		if word == "" {
			continue
		}
		if isAllUpper(word) {
			buf.WriteString(word)
			continue
		}
		runes := []rune(strings.ToLower(word))
		runes[0] = unicode.ToUpper(runes[0])
		buf.WriteString(string(runes))
	}
	return buf.String()
}

func splitWords(s string) []string {
	if s == "" {
		return nil
	}
	runes := []rune(s)
	var parts []string
	current := []rune{runes[0]}
	for i := 1; i < len(runes); i++ {
		r := runes[i]
		prev := runes[i-1]
		nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
		switch {
		case unicode.IsUpper(r):
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				parts = append(parts, string(current))
				current = []rune{r}
			} else {
				current = append(current, r)
			}
		case unicode.IsDigit(r):
			if !unicode.IsDigit(prev) {
				parts = append(parts, string(current))
				current = []rune{r}
			} else {
				current = append(current, r)
			}
		default:
			if unicode.IsDigit(prev) {
				parts = append(parts, string(current))
				current = []rune{r}
			} else {
				current = append(current, r)
			}
		}
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	return parts
}

func isAllUpper(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func findMatchingBrace(src string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(src) || src[openIdx] != '{' {
		return -1
	}
	depth := 0
	for i := openIdx; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
