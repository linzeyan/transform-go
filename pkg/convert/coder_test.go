package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeContent(t *testing.T) {
	res, err := EncodeContent("hi")
	require.NoError(t, err)
	require.Equal(t, "NBUQ====", res[EncodingBase32Std])
	require.Equal(t, "NBUQ", res[EncodingBase32StdNoPadding])
	require.Equal(t, "D1KG====", res[EncodingBase32Hex])
	require.Equal(t, "D1KG", res[EncodingBase32HexNoPadding])
	require.Equal(t, "aGk=", res[EncodingBase64Std])
	require.Equal(t, "aGk", res[EncodingBase64RawStd])
	require.Equal(t, "aGk=", res[EncodingBase64URL])
	require.Equal(t, "aGk", res[EncodingBase64RawURL])
	require.Equal(t, "BP@", res[EncodingBase85ASCII])
	require.Equal(t, "qaD", res[EncodingBase91])
	require.Equal(t, "6869", res[EncodingHexUpper])
}

func TestDecodeContent(t *testing.T) {
	type testCase struct {
		kind    string
		encoded string
		expect  string
	}
	cases := []testCase{
		{EncodingBase32Std, "NBUQ====", "hi"},
		{EncodingBase32HexNoPadding, "D1KG", "hi"},
		{EncodingBase64URL, "aGk=", "hi"},
		{EncodingBase64RawURL, "aGk", "hi"},
		{EncodingBase85ASCII, "BP@", "hi"},
		{EncodingBase91, "qaD", "hi"},
		{EncodingHexUpper, "6869", "hi"},
	}
	for _, tc := range cases {
		result, err := DecodeContent(tc.kind, tc.encoded)
		require.NoError(t, err, tc.kind)
		require.Equal(t, tc.expect, result, tc.kind)
	}
	_, err := DecodeContent("unknown", "hi")
	require.Error(t, err)
	_, err = DecodeContent(EncodingBase32Std, "invalid===")
	require.Error(t, err)
}

func TestHashContent(t *testing.T) {
	res := HashContent("hello")
	require.Equal(t, "5d41402abc4b2a76b9719d911017c592", res["md5"])
	require.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", res["sha256"])
	require.Equal(t, "3610a686", res["crc32_ieee"])
	require.Equal(t, "a430d84680aabd0b", res["fnv64a"])
}

func TestURLEncodeDecode(t *testing.T) {
	input := "https://example.com/search?q=中文 空格&x=1+2"
	encoded := URLEncode(input)
	require.Equal(t, "https%3A%2F%2Fexample.com%2Fsearch%3Fq%3D%E4%B8%AD%E6%96%87+%E7%A9%BA%E6%A0%BC%26x%3D1%2B2", encoded)
	decoded, err := URLDecode(encoded)
	require.NoError(t, err)
	require.Equal(t, input, decoded)
}

func TestJWTEncodeDecode(t *testing.T) {
	payload := `{"sub":"1234567890","name":"John Doe","admin":true}`
	token, err := JWTEncode(payload, "secret", "HS256")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parts, err := JWTDecode(token)
	require.NoError(t, err)
	require.Equal(t, "HS256", parts.Algorithm)
	require.Contains(t, parts.Payload, `"John Doe"`)
	require.Contains(t, parts.Payload, `"admin": true`)
	require.Contains(t, parts.Header, `"typ": "JWT"`)
	require.NotEmpty(t, parts.Signature)
}
