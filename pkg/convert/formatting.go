package convert

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"go/format"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ugorji/go/codec"
)

const (
	formatJSON     = "JSON"
	formatGoStruct = "Go Struct"
	formatYAML     = "YAML"
	formatTOML     = "TOML"
	formatXML      = "XML"
	formatSchema   = "JSON Schema"
	formatGraphQL  = "GraphQL Schema"
	formatProtobuf = "Protobuf"
	formatTOON     = "TOON"
	formatMsgPack  = "MsgPack"
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
	formatXML: {
		ToJSON:   XMLToJSON,
		FromJSON: JSONToXML,
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
	formatTOON: {
		ToJSON:   TOONToJSON,
		FromJSON: JSONToTOON,
	},
	formatMsgPack: {
		ToJSON:   MsgPackToJSON,
		FromJSON: JSONToMsgPack,
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
	case formatXML:
		if minify {
			return compactXML(input)
		}
		jsonStr, err := XMLToJSON(input)
		if err != nil {
			return "", err
		}
		return JSONToXML(jsonStr)
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

func compactXML(src string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(src))
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			if err := encoder.EncodeToken(xml.CharData([]byte(text))); err != nil {
				return "", err
			}
		default:
			if err := encoder.EncodeToken(tok); err != nil {
				return "", err
			}
		}
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var msgpackHandle codec.MsgpackHandle

func init() {
	msgpackHandle.RawToString = true
}

// JSONToMsgPack encodes JSON into MsgPack and returns a base64 string.
func JSONToMsgPack(input string) (string, error) {
	var data any
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return "", err
	}
	buf := make([]byte, 0, 512)
	enc := codec.NewEncoderBytes(&buf, &msgpackHandle)
	if err := enc.Encode(data); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// MsgPackToJSON decodes a base64 MsgPack payload into pretty JSON.
func MsgPackToJSON(input string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}
	var data any
	dec := codec.NewDecoderBytes(raw, &msgpackHandle)
	if err := dec.Decode(&data); err != nil {
		return "", err
	}
	data = normalizeMsgPackValue(data)
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(pretty), nil
}

func normalizeMsgPackValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			val[k] = normalizeMsgPackValue(inner)
		}
		return val
	case map[interface{}]interface{}:
		out := make(map[string]any, len(val))
		for k, inner := range val {
			out[fmt.Sprint(k)] = normalizeMsgPackValue(inner)
		}
		return out
	case []any:
		for i, inner := range val {
			val[i] = normalizeMsgPackValue(inner)
		}
		return val
	default:
		return val
	}
}

const toonIndent = "  "

// JSONToTOON encodes JSON into TOON text.
func JSONToTOON(input string) (string, error) {
	var data any
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return "", err
	}
	builder := &strings.Builder{}
	if err := writeTOON(builder, "", data, 0, ','); err != nil {
		return "", err
	}
	return strings.TrimRight(builder.String(), "\n"), nil
}

// TOONToJSON decodes TOON text back into JSON.
func TOONToJSON(input string) (string, error) {
	parser := newToonParser(input)
	value, err := parser.parse()
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func writeTOON(b *strings.Builder, key string, value any, depth int, docDelim rune) error {
	switch v := value.(type) {
	case map[string]any:
		if key != "" {
			writeIndent(b, depth)
			fmt.Fprintf(b, "%s:\n", key)
			depth++
		}
		keys := orderedKeys(v)
		for _, k := range keys {
			if err := writeTOON(b, k, v[k], depth, docDelim); err != nil {
				return err
			}
		}
	case []any:
		return writeTOONArray(b, key, v, depth, docDelim)
	default:
		writeIndent(b, depth)
		if key != "" {
			fmt.Fprintf(b, "%s: %s\n", key, formatPrimitive(v, docDelim))
		} else {
			fmt.Fprintf(b, "%s\n", formatPrimitive(v, docDelim))
		}
	}
	return nil
}

func writeTOONArray(b *strings.Builder, key string, arr []any, depth int, docDelim rune) error {
	length := len(arr)
	fields, rows, ok := detectTabular(arr)
	indent := strings.Repeat(toonIndent, depth)
	delimiter := ','
	if key == "" {
		if ok {
			fmt.Fprintf(b, "%s[%d]{%s}:\n", indent, length, strings.Join(fields, ","))
		} else if allPrimitives(arr) {
			vals := joinPrimitiveArray(arr, delimiter)
			fmt.Fprintf(b, "%s[%d]: %s\n", indent, length, vals)
		} else {
			fmt.Fprintf(b, "%s[%d]:\n", indent, length)
		}
	} else {
		if ok {
			fmt.Fprintf(b, "%s%s[%d]{%s}:\n", indent, key, length, strings.Join(fields, ","))
		} else if allPrimitives(arr) {
			vals := joinPrimitiveArray(arr, delimiter)
			fmt.Fprintf(b, "%s%s[%d]: %s\n", indent, key, length, vals)
		} else {
			fmt.Fprintf(b, "%s%s[%d]:\n", indent, key, length)
		}
	}

	if ok {
		for _, row := range rows {
			writeIndent(b, depth+1)
			values := make([]string, len(fields))
			for idx, field := range fields {
				values[idx] = formatPrimitive(row[field], delimiter)
			}
			fmt.Fprintf(b, "%s\n", strings.Join(values, ","))
		}
		return nil
	}

	if allPrimitives(arr) {
		return nil
	}
	return writeListEntries(b, arr, depth, docDelim)
}

func joinPrimitiveArray(arr []any, delim rune) string {
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = formatPrimitive(v, delim)
	}
	sep := string(delim)
	return strings.Join(parts, sep)
}

func writeListEntries(b *strings.Builder, items []any, depth int, docDelim rune) error {
	for _, item := range items {
		writeIndent(b, depth+1)
		switch val := item.(type) {
		case map[string]any:
			if err := writeListObject(b, val, depth, docDelim); err != nil {
				return err
			}
		case []any:
			if err := writeArrayListItem(b, val, depth, docDelim); err != nil {
				return err
			}
		default:
			fmt.Fprintf(b, "- %s\n", formatPrimitive(val, docDelim))
		}
	}
	return nil
}

func writeListObject(b *strings.Builder, obj map[string]any, depth int, docDelim rune) error {
	fmt.Fprint(b, "-")
	if len(obj) == 0 {
		fmt.Fprint(b, "\n")
		return nil
	}
	keys := orderedKeys(obj)
	first := true
	for _, k := range keys {
		val := obj[k]
		if first {
			fmt.Fprint(b, " ")
			if err := writeInlineField(b, k, val, depth, docDelim); err != nil {
				return err
			}
			first = false
			continue
		}
		fmt.Fprint(b, "\n")
		if err := writeTOON(b, k, val, depth+2, docDelim); err != nil {
			return err
		}
	}
	fmt.Fprint(b, "\n")
	return nil
}

func writeInlineField(b *strings.Builder, key string, value any, depth int, docDelim rune) error {
	switch val := value.(type) {
	case map[string]any:
		fmt.Fprintf(b, "%s:\n", key)
		return writeTOON(b, "", val, depth+2, docDelim)
	case []any:
		return writeFieldArrayInline(b, key, val, depth+1, docDelim)
	default:
		fmt.Fprintf(b, "%s: %s", key, formatPrimitive(val, docDelim))
		return nil
	}
}

func writeFieldArrayInline(b *strings.Builder, key string, arr []any, depth int, docDelim rune) error {
	fields, rows, ok := detectTabular(arr)
	delimiter := ','
	if ok {
		fmt.Fprintf(b, "%s[%d]{%s}:\n", key, len(arr), strings.Join(fields, ","))
		for _, row := range rows {
			writeIndent(b, depth+1)
			values := make([]string, len(fields))
			for idx, field := range fields {
				values[idx] = formatPrimitive(row[field], delimiter)
			}
			fmt.Fprintf(b, "%s\n", strings.Join(values, ","))
		}
		return nil
	}
	if allPrimitives(arr) {
		fmt.Fprintf(b, "%s[%d]: %s", key, len(arr), joinPrimitiveArray(arr, delimiter))
		return nil
	}
	fmt.Fprintf(b, "%s[%d]:\n", key, len(arr))
	return writeListEntries(b, arr, depth, docDelim)
}

func writeArrayListItem(b *strings.Builder, arr []any, depth int, docDelim rune) error {
	fields, rows, ok := detectTabular(arr)
	delimiter := ','
	if ok {
		fmt.Fprintf(b, "- [%d]{%s}:\n", len(arr), strings.Join(fields, ","))
		for _, row := range rows {
			writeIndent(b, depth+2)
			values := make([]string, len(fields))
			for idx, field := range fields {
				values[idx] = formatPrimitive(row[field], delimiter)
			}
			fmt.Fprintf(b, "%s\n", strings.Join(values, ","))
		}
		return nil
	}
	if allPrimitives(arr) {
		fmt.Fprintf(b, "- [%d]: %s\n", len(arr), joinPrimitiveArray(arr, delimiter))
		return nil
	}
	fmt.Fprintf(b, "- [%d]:\n", len(arr))
	return writeListEntries(b, arr, depth+1, docDelim)
}

func orderedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func allPrimitives(arr []any) bool {
	for _, v := range arr {
		switch v.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

func detectTabular(arr []any) ([]string, []map[string]any, bool) {
	if len(arr) == 0 {
		return nil, nil, false
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return nil, nil, false
	}
	fields := orderedKeys(first)
	rows := make([]map[string]any, 0, len(arr))
	rows = append(rows, first)
	for i := 1; i < len(arr); i++ {
		obj, ok := arr[i].(map[string]any)
		if !ok {
			return nil, nil, false
		}
		if !sameFieldSet(fields, obj) {
			return nil, nil, false
		}
		rows = append(rows, obj)
	}
	for _, row := range rows {
		for _, f := range fields {
			if _, ok := row[f]; !ok {
				return nil, nil, false
			}
			switch row[f].(type) {
			case map[string]any, []any:
				return nil, nil, false
			}
		}
	}
	return fields, rows, true
}

func sameFieldSet(fields []string, obj map[string]any) bool {
	if len(fields) != len(obj) {
		return false
	}
	for _, f := range fields {
		if _, ok := obj[f]; !ok {
			return false
		}
	}
	return true
}

func writeIndent(b *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		b.WriteString(toonIndent)
	}
}

var numberPattern = regexp.MustCompile(`^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$`)

func formatPrimitive(value any, delim rune) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		if needsQuote(v, delim) {
			return quoteString(v)
		}
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func needsQuote(s string, delim rune) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return true
	}
	switch s {
	case "true", "false", "null":
		return true
	}
	if numberPattern.MatchString(s) {
		return true
	}
	if strings.ContainsAny(s, ":\"\\[]{}") {
		return true
	}
	if strings.ContainsRune(s, '\n') || strings.ContainsRune(s, '\r') || strings.ContainsRune(s, '\t') {
		return true
	}
	if strings.ContainsRune(s, delim) {
		return true
	}
	if strings.HasPrefix(s, "-") {
		return true
	}
	return false
}

func quoteString(s string) string {
	replacer := strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	)
	return `"` + replacer.Replace(s) + `"`
}

// --------- Minimal parser ----------

type toonParser struct {
	lines []toonLine
	idx   int
}

type toonLine struct {
	depth  int
	text   string
	number int
}

func newToonParser(input string) *toonParser {
	raw := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	lines := make([]toonLine, 0, len(raw))
	for i, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		depth := countIndent(line)
		text := strings.TrimSpace(line)
		lines = append(lines, toonLine{depth: depth, text: text, number: i + 1})
	}
	return &toonParser{lines: lines}
}

func countIndent(line string) int {
	count := 0
	for strings.HasPrefix(line[count:], "  ") {
		count += 2
	}
	return count / 2
}

func (p *toonParser) parse() (any, error) {
	if len(p.lines) == 0 {
		return map[string]any{}, nil
	}
	line := p.lines[0]
	if strings.HasPrefix(line.text, "[") {
		return p.parseHeader(0)
	}
	if !strings.Contains(line.text, ":") && len(p.lines) == 1 {
		return parsePrimitiveToken(line.text), nil
	}
	return p.parseObject(0)
}

func (p *toonParser) parseObject(depth int) (map[string]any, error) {
	result := map[string]any{}
	for p.idx < len(p.lines) {
		line := p.lines[p.idx]
		if line.depth < depth {
			break
		}
		if line.depth > depth {
			return nil, fmt.Errorf("unexpected indentation near line %d", line.number)
		}
		if strings.HasPrefix(line.text, "[") {
			break
		}
		key, rest, ok := strings.Cut(line.text, ":")
		if !ok {
			return nil, fmt.Errorf("expected key on line %d", line.number)
		}
		key = strings.TrimSpace(key)
		value := strings.TrimSpace(rest)
		p.idx++
		switch {
		case headerRegex.MatchString(line.text):
			p.idx--
			arr, err := p.parseHeader(depth)
			if err != nil {
				return nil, err
			}
			result[key] = arr
		case value == "":
			nested, err := p.parseObject(depth + 1)
			if err != nil {
				return nil, err
			}
			result[key] = nested
		default:
			result[key] = parsePrimitiveToken(value)
		}
	}
	return result, nil
}

var headerRegex = regexp.MustCompile(`^[A-Za-z0-9._"]*\[\d+[|\t]?\](?:\{.*\})?:`)

func (p *toonParser) parseHeader(depth int) (any, error) {
	line := p.lines[p.idx]
	p.idx++
	text := line.text
	beforeColon, inline, _ := strings.Cut(text, ":")
	inline = strings.TrimSpace(inline)
	header := beforeColon
	bracketStart := strings.Index(header, "[")
	if bracketStart == -1 {
		return nil, fmt.Errorf("invalid header on line %d", line.number)
	}
	bracket := header[bracketStart:]
	brace := ""
	if idx := strings.Index(bracket, "{"); idx != -1 {
		brace = bracket[idx:]
		bracket = bracket[:idx]
	}
	lengthText := strings.Trim(bracket, "[]\t|")
	length, err := strconv.Atoi(lengthText)
	if err != nil {
		return nil, fmt.Errorf("invalid length on line %d", line.number)
	}
	delimiter := ','
	if strings.Contains(header, "\t") {
		delimiter = '\t'
	} else if strings.Contains(header, "|") {
		delimiter = '|'
	}
	if inline != "" {
		values := splitDelimited(inline, delimiter)
		arr := make([]any, 0, len(values))
		for _, v := range values {
			arr = append(arr, parsePrimitiveToken(v))
		}
		return arr, nil
	}
	if brace != "" {
		fieldList := strings.Trim(brace, "{}")
		fields := splitDelimited(fieldList, delimiter)
		rows := make([]map[string]any, 0, length)
		for i := 0; i < length && p.idx < len(p.lines); i++ {
			rowLine := p.lines[p.idx]
			if rowLine.depth != depth+1 {
				return nil, fmt.Errorf("expected row at line %d", rowLine.number)
			}
			values := splitDelimited(rowLine.text, delimiter)
			if len(values) != len(fields) {
				return nil, fmt.Errorf("row width mismatch near line %d", rowLine.number)
			}
			row := map[string]any{}
			for idx, field := range fields {
				row[field] = parsePrimitiveToken(values[idx])
			}
			rows = append(rows, row)
			p.idx++
		}
		arr := make([]any, len(rows))
		for i, row := range rows {
			arr[i] = row
		}
		return arr, nil
	}
	items := make([]any, 0, length)
	for p.idx < len(p.lines) {
		itemLine := p.lines[p.idx]
		if itemLine.depth < depth+1 {
			break
		}
		if itemLine.depth > depth+1 {
			return nil, fmt.Errorf("unexpected indentation in list near line %d", itemLine.number)
		}
		if !strings.HasPrefix(itemLine.text, "-") {
			break
		}
		content := strings.TrimSpace(strings.TrimPrefix(itemLine.text, "-"))
		p.idx++
		if content == "" {
			items = append(items, map[string]any{})
			continue
		}
		if strings.Contains(content, ":") {
			subKey, rest, _ := strings.Cut(content, ":")
			subKey = strings.TrimSpace(subKey)
			rest = strings.TrimSpace(rest)
			if rest == "" {
				obj, err := p.parseObject(depth + 2)
				if err != nil {
					return nil, err
				}
				items = append(items, map[string]any{subKey: obj})
			} else {
				items = append(items, map[string]any{subKey: parsePrimitiveToken(rest)})
			}
		} else if headerRegex.MatchString(content) {
			p.idx--
			arr, err := p.parseHeader(depth + 1)
			if err != nil {
				return nil, err
			}
			items = append(items, arr)
		} else {
			items = append(items, parsePrimitiveToken(content))
		}
	}
	return items, nil
}

func splitDelimited(input string, delim rune) []string {
	var result []string
	current := strings.Builder{}
	inQuotes := false
	escaped := false
	for _, ch := range input {
		switch {
		case escaped:
			current.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '"':
			inQuotes = !inQuotes
			current.WriteRune(ch)
		case ch == delim && !inQuotes:
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

func parsePrimitiveToken(token string) any {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"") {
		unquoted, err := strconv.Unquote(token)
		if err == nil {
			return unquoted
		}
		return token
	}
	switch token {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if numberPattern.MatchString(token) {
		if strings.ContainsAny(token, ".eE") {
			f, err := strconv.ParseFloat(token, 64)
			if err == nil {
				return f
			}
		} else {
			i, err := strconv.ParseInt(token, 10, 64)
			if err == nil {
				return i
			}
		}
	}
	return token
}
