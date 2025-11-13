package convert

import (
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"net"
	"regexp"
	"strconv"
	"strings"
)

type NumberBases struct {
	Binary  string `json:"binary"`
	Octal   string `json:"octal"`
	Decimal string `json:"decimal"`
	Hex     string `json:"hex"`
}

type IPv4Result struct {
	Type       string `json:"type"`
	Input      string `json:"input"`
	CIDR       string `json:"cidr,omitempty"`
	Mask       string `json:"mask,omitempty"`
	RangeStart string `json:"rangeStart,omitempty"`
	RangeEnd   string `json:"rangeEnd,omitempty"`
	Total      string `json:"total,omitempty"`
	Standard   string `json:"standard,omitempty"`
	ThreePart  string `json:"threePart,omitempty"`
	TwoPart    string `json:"twoPart,omitempty"`
	Integer    string `json:"integer,omitempty"`
}

func ConvertNumberBase(base, value string) (NumberBases, error) {
	result := NumberBases{}
	num, err := parseNumberByBase(base, value)
	if err != nil {
		return result, err
	}
	result.Binary = formatBigInt(num, 2)
	result.Octal = formatBigInt(num, 8)
	result.Decimal = formatBigInt(num, 10)
	result.Hex = strings.ToUpper(formatBigInt(num, 16))
	return result, nil
}

func parseNumberByBase(base, value string) (*big.Int, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil, errors.New("value is empty")
	}
	clean = strings.ReplaceAll(clean, "_", "")
	var radix int
	switch base {
	case "binary":
		radix = 2
		clean = strings.TrimPrefix(strings.TrimPrefix(clean, "0b"), "0B")
	case "octal":
		radix = 8
		clean = strings.TrimPrefix(strings.TrimPrefix(clean, "0o"), "0O")
	case "decimal":
		radix = 10
	case "hex":
		radix = 16
		if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
			clean = clean[2:]
		}
	default:
		return nil, fmt.Errorf("unsupported base %s", base)
	}
	intVal := new(big.Int)
	if len(clean) == 0 {
		return nil, errors.New("value is empty")
	}
	clean, _ = strings.CutPrefix(clean, "+")
	if _, ok := intVal.SetString(clean, radix); !ok {
		return nil, fmt.Errorf("invalid %s value", base)
	}
	return intVal, nil
}

func formatBigInt(num *big.Int, base int) string {
	if num == nil {
		return ""
	}
	return num.Text(base)
}

func IPv4Info(input string) (IPv4Result, error) {
	trimmed := strings.TrimSpace(input)
	res := IPv4Result{Input: trimmed}
	if trimmed == "" {
		return res, errors.New("input is empty")
	}
	if looksLikeRange(trimmed) {
		return ipv4Range(trimmed)
	}
	if strings.Contains(trimmed, "/") {
		return ipv4WithPrefix(trimmed)
	}
	ip := parseIPv4(trimmed)
	if ip == nil {
		return res, fmt.Errorf("invalid IPv4 address: %s", trimmed)
	}
	octets := []byte(ip)
	res.Type = "single"
	res.Standard = trimmed
	res.ThreePart = fmt.Sprintf("%d.%d.%d", octets[0], octets[1], int(octets[2])<<8|int(octets[3]))
	res.TwoPart = fmt.Sprintf("%d.%d", octets[0], int(octets[1])<<16|int(octets[2])<<8|int(octets[3]))
	res.Integer = strconv.FormatUint(uint64(ipToUint32(ip)), 10)
	res.CIDR = fmt.Sprintf("%s/32", trimmed)
	res.RangeStart = trimmed
	res.RangeEnd = trimmed
	res.Total = "1"
	return res, nil
}

func ipv4WithPrefix(input string) (IPv4Result, error) {
	res := IPv4Result{Input: input, Type: "network"}
	parts := strings.SplitN(input, "/", 2)
	ip := parseIPv4(strings.TrimSpace(parts[0]))
	if ip == nil {
		return res, fmt.Errorf("invalid IPv4 address: %s", parts[0])
	}
	var prefix int
	var mask net.IPMask
	right := strings.TrimSpace(parts[1])
	if strings.Contains(right, ".") {
		maskIP := parseIPv4(right)
		if maskIP == nil {
			return res, fmt.Errorf("invalid subnet mask: %s", right)
		}
		mask = net.IPMask(maskIP)
		var err error
		if prefix, err = maskSize(mask); err != nil {
			return res, err
		}
	} else {
		val, err := strconv.Atoi(right)
		if err != nil || val < 0 || val > 32 {
			return res, fmt.Errorf("invalid prefix length: %s", right)
		}
		prefix = val
		mask = net.CIDRMask(prefix, 32)
	}
	ipInt := ipToUint32(ip)
	maskInt := maskToUint32(mask)
	network := ipInt & maskInt
	broadcast := network | ^maskInt
	res.CIDR = fmt.Sprintf("%s/%d", uint32ToIP(network).String(), prefix)
	res.Mask = net.IP(mask).String()
	res.RangeStart = uint32ToIP(network).String()
	res.RangeEnd = uint32ToIP(broadcast).String()
	hostBits := 32 - prefix
	if hostBits >= 0 && hostBits <= 32 {
		total := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(hostBits)), nil)
		res.Total = total.String()
	}
	return res, nil
}

func ipv4Range(input string) (IPv4Result, error) {
	res := IPv4Result{Input: input, Type: "range"}
	normalized := strings.NewReplacer(" ", "", "->", "-", "—", "-", "–", "-").Replace(input)
	parts := strings.Split(normalized, "-")
	if len(parts) != 2 {
		return res, errors.New("range must be in start-end format")
	}
	startIP := parseIPv4(parts[0])
	endIP := parseIPv4(parts[1])
	if startIP == nil || endIP == nil {
		return res, errors.New("invalid IPv4 range")
	}
	start := ipToUint32(startIP)
	end := ipToUint32(endIP)
	if start > end {
		return res, errors.New("start IP must be less than or equal to end IP")
	}
	res.RangeStart = startIP.String()
	res.RangeEnd = endIP.String()
	total := new(big.Int).SetUint64(uint64(end - start))
	total = total.Add(total, big.NewInt(1))
	res.Total = total.String()
	res.CIDR = strings.Join(ipRangeToCIDRs(start, end), ", ")
	return res, nil
}

func parseIPv4(value string) net.IP {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	return ip
}

func ipToUint32(ip net.IP) uint32 {
	return binaryToUint32(ip[0], ip[1], ip[2], ip[3])
}

func uint32ToIP(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func maskToUint32(mask net.IPMask) uint32 {
	if len(mask) != 4 {
		return 0
	}
	return binaryToUint32(mask[0], mask[1], mask[2], mask[3])
}

func binaryToUint32(a, b, c, d byte) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)
}

func maskSize(mask net.IPMask) (int, error) {
	ones, bits := mask.Size()
	if bits != 32 {
		return 0, errors.New("invalid mask size")
	}
	return ones, nil
}

func MarkdownToHTML(input string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	var builder strings.Builder
	inList := false
	inCodeBlock := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			if inCodeBlock {
				builder.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				builder.WriteString("<pre><code>")
				inCodeBlock = true
			}
			continue
		}
		if inCodeBlock {
			builder.WriteString(htmlEscape(line))
			builder.WriteString("\n")
			continue
		}
		if trim == "" {
			if inList {
				builder.WriteString("</ul>\n")
				inList = false
			}
			continue
		}
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") {
			if !inList {
				builder.WriteString("<ul>\n")
				inList = true
			}
			item := strings.TrimSpace(trim[2:])
			builder.WriteString("<li>")
			builder.WriteString(applyInlineMarkdown(item))
			builder.WriteString("</li>\n")
			continue
		}
		if inList {
			builder.WriteString("</ul>\n")
			inList = false
		}
		if headingLevel := markdownHeadingLevel(trim); headingLevel > 0 {
			content := strings.TrimSpace(trim[headingLevel:])
			builder.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", headingLevel, applyInlineMarkdown(content), headingLevel))
			continue
		}
		if i != len(lines)-1 && strings.TrimSpace(lines[i+1]) == "" {
			builder.WriteString("<p>")
			builder.WriteString(applyInlineMarkdown(trim))
			builder.WriteString("</p>\n")
		} else {
			builder.WriteString(applyInlineMarkdown(trim))
			builder.WriteString("\n")
		}
	}
	if inList {
		builder.WriteString("</ul>\n")
	}
	if inCodeBlock {
		builder.WriteString("</code></pre>\n")
	}
	return builder.String(), nil
}

func markdownHeadingLevel(line string) int {
	count := 0
	for _, ch := range line {
		if ch == '#' {
			count++
		} else {
			break
		}
	}
	if count == 0 {
		return 0
	}
	if count > 6 {
		count = 6
	}
	return count
}

func applyInlineMarkdown(text string) string {
	out := htmlEscape(text)
	out = replaceDelimited(out, "**", "<strong>", "</strong>")
	out = replaceDelimited(out, "*", "<em>", "</em>")
	out = replaceDelimited(out, "`", "<code>", "</code>")
	out = convertMarkdownLinks(out)
	return out
}

func replaceDelimited(text, marker, openTag, closeTag string) string {
	var builder strings.Builder
	parts := strings.Split(text, marker)
	for i, part := range parts {
		builder.WriteString(part)
		if i == len(parts)-1 {
			break
		}
		if i%2 == 0 {
			builder.WriteString(openTag)
		} else {
			builder.WriteString(closeTag)
		}
	}
	return builder.String()
}

func convertMarkdownLinks(text string) string {
	result := text
	const open = "["
	const mid = "]("
	for {
		start := strings.Index(result, open)
		if start == -1 {
			break
		}
		middle := strings.Index(result[start:], mid)
		if middle == -1 {
			break
		}
		middle += start
		end := strings.Index(result[middle:], ")")
		if end == -1 {
			break
		}
		end += middle
		label := result[start+1 : middle]
		href := result[middle+2 : end]
		replacement := fmt.Sprintf(`<a href="%s">%s</a>`, htmlEscape(href), htmlEscape(label))
		result = result[:start] + replacement + result[end+1:]
	}
	return result
}

func stripTags(text string) string {
	result := text
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		end += start
		result = result[:start] + result[end+1:]
	}
	return strings.TrimSpace(result)
}

func htmlEscape(input string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(input)
}

func htmlUnescape(input string) string {
	replacer := strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
	)
	return replacer.Replace(input)
}

func looksLikeRange(input string) bool {
	normalized := strings.ReplaceAll(input, " ", "")
	return strings.Contains(normalized, "-") || strings.Contains(normalized, "->")
}

func ipRangeToCIDRs(start, end uint32) []string {
	var cidrs []string
	for start <= end {
		maxSize := start & ^(start - 1)
		if maxSize == 0 {
			maxSize = 1 << 31
		}
		remaining := end - start + 1
		size := maxSize
		for size > remaining {
			size >>= 1
		}
		prefix := 33 - bits.Len32(size)
		if prefix < 0 {
			prefix = 0
		}
		cidrs = append(cidrs, fmt.Sprintf("%s/%d", uint32ToIP(start).String(), prefix))
		start += size
	}
	return cidrs
}

var (
	reScript     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reHeading    = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	reParagraph  = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	reDiv        = regexp.MustCompile(`(?is)<div[^>]*>(.*?)</div>`)
	reListItem   = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	reStrong     = regexp.MustCompile(`(?is)<(?:strong|b)[^>]*>(.*?)</(?:strong|b)>`)
	reEm         = regexp.MustCompile(`(?is)<(?:em|i)[^>]*>(.*?)</(?:em|i)>`)
	reCodeBlock  = regexp.MustCompile(`(?is)<pre[^>]*><code[^>]*>(.*?)</code></pre>`)
	reInlineCode = regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	reLink       = regexp.MustCompile(`(?is)<a[^>]*href=["'](.*?)["'][^>]*>(.*?)</a>`)
	reBreak      = regexp.MustCompile(`(?is)<br\s*/?>`)
	reTag        = regexp.MustCompile(`(?is)<[^>]+>`)
)

func HTMLToMarkdown(input string) (string, error) {
	text := strings.ReplaceAll(input, "\r\n", "\n")
	text = reScript.ReplaceAllString(text, "")
	text = reStyle.ReplaceAllString(text, "")
	text = reCodeBlock.ReplaceAllStringFunc(text, func(match string) string {
		sub := reCodeBlock.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		content := strings.TrimSpace(htmlUnescape(sub[1]))
		return "```\n" + content + "\n```\n\n"
	})
	text = reBreak.ReplaceAllString(text, "\n")
	text = reHeading.ReplaceAllStringFunc(text, func(match string) string {
		sub := reHeading.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		level, _ := strconv.Atoi(sub[1])
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		content := strings.TrimSpace(htmlUnescape(sub[2]))
		return strings.Repeat("#", level) + " " + content + "\n\n"
	})
	text = reParagraph.ReplaceAllString(text, "\n$1\n\n")
	text = reDiv.ReplaceAllString(text, "\n$1\n")
	text = reListItem.ReplaceAllString(text, "\n- $1")
	text = strings.ReplaceAll(text, "</ul>", "\n\n")
	text = strings.ReplaceAll(text, "</ol>", "\n\n")
	text = reStrong.ReplaceAllString(text, "**$1**")
	text = reEm.ReplaceAllString(text, "*$1*")
	text = reInlineCode.ReplaceAllString(text, "`$1`")
	text = reLink.ReplaceAllStringFunc(text, func(match string) string {
		sub := reLink.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		href := htmlUnescape(sub[1])
		label := htmlUnescape(stripTags(sub[2]))
		return fmt.Sprintf("[%s](%s)", label, href)
	})
	text = reTag.ReplaceAllString(text, "")
	lines := strings.Split(htmlUnescape(text), "\n")
	var compact []string
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			if len(compact) > 0 && compact[len(compact)-1] == "" {
				continue
			}
			compact = append(compact, "")
			continue
		}
		compact = append(compact, line)
	}
	return strings.TrimSpace(strings.Join(compact, "\n")), nil
}
