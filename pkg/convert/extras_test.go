package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertNumberBase(t *testing.T) {
	res, err := ConvertNumberBase("decimal", "255")
	require.NoError(t, err)
	require.Equal(t, "11111111", res.Binary)
	require.Equal(t, "377", res.Octal)
	require.Equal(t, "255", res.Decimal)
	require.Equal(t, "FF", res.Hex)

	res, err = ConvertNumberBase("hex", "0xff")
	require.NoError(t, err)
	require.Equal(t, "255", res.Decimal)

	_, err = ConvertNumberBase("binary", "2")
	require.Error(t, err)
}

func TestMarkdownHTML(t *testing.T) {
	html, err := MarkdownToHTML("# Title\n\n- item")
	require.NoError(t, err)
	require.Contains(t, html, "<h1>")
	require.Contains(t, html, "<ul>")

	md, err := HTMLToMarkdown("<h1>Title</h1><p>Hello <strong>world</strong></p>")
	require.NoError(t, err)
	require.Contains(t, md, "# Title")
	require.Contains(t, md, "**world**")

	md2, err := HTMLToMarkdown("<HTML><body><h2 class=\"x\">Hi</h2><p>Line<br/>Next</p></body></HTML>")
	require.NoError(t, err)
	require.Contains(t, md2, "## Hi")
	require.Contains(t, md2, "Line")
}

func TestIPv4Info(t *testing.T) {
	res, err := IPv4Info("1.1.1.1")
	require.NoError(t, err)
	require.Equal(t, "single", res.Type)
	require.Equal(t, "1.1.257", res.ThreePart)
	require.Equal(t, "1.65793", res.TwoPart)
	require.Equal(t, "16843009", res.Integer)

	res, err = IPv4Info("192.168.0.0/24")
	require.NoError(t, err)
	require.Equal(t, "network", res.Type)
	require.Equal(t, "192.168.0.0", res.RangeStart)
	require.Equal(t, "192.168.0.255", res.RangeEnd)
	require.Equal(t, "256", res.Total)

	res, err = IPv4Info("10.0.0.0/255.255.0.0")
	require.NoError(t, err)
	require.Equal(t, "10.0.0.0", res.RangeStart)
	require.Equal(t, "10.0.255.255", res.RangeEnd)
	require.Equal(t, "65536", res.Total)

	res, err = IPv4Info("1.1.1.0 - 1.1.1.255")
	require.NoError(t, err)
	require.Equal(t, "range", res.Type)
	require.Equal(t, "256", res.Total)
	require.Contains(t, res.CIDR, "/24")
}
