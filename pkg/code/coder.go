package code

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/ascii85"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"hash/fnv"
	"io"
	"net/url"
	"strings"
)

const (
	EncodingBase32Std          = "base32_standard"
	EncodingBase32StdNoPadding = "base32_standard_no_padding"
	EncodingBase32Hex          = "base32_hex"
	EncodingBase32HexNoPadding = "base32_hex_no_padding"
	EncodingBase64Std          = "base64_standard"
	EncodingBase64RawStd       = "base64_raw_standard"
	EncodingBase64URL          = "base64_url"
	EncodingBase64RawURL       = "base64_raw_url"
	EncodingBase85ASCII        = "base85_ascii85"
	EncodingBase91             = "base91"
	EncodingHexUpper           = "hex_upper"
)

var (
	base32StdNoPadding = base32.StdEncoding.WithPadding(base32.NoPadding)
	base32HexNoPadding = base32.HexEncoding.WithPadding(base32.NoPadding)
	base64RawStd       = base64.StdEncoding.WithPadding(base64.NoPadding)
	base64RawURL       = base64.URLEncoding.WithPadding(base64.NoPadding)
	crc32Castagnoli    = crc32.MakeTable(crc32.Castagnoli)
	crc64ISOTable      = crc64.MakeTable(crc64.ISO)
	crc64ECMATable     = crc64.MakeTable(crc64.ECMA)
	base91Lookup       = initBase91Lookup()
)

// EncodeContent runs through all supported encodings and returns every representation.
func EncodeContent(input string) (map[string]string, error) {
	data := []byte(input)
	out := map[string]string{
		EncodingBase32Std:          base32.StdEncoding.EncodeToString(data),
		EncodingBase32StdNoPadding: base32StdNoPadding.EncodeToString(data),
		EncodingBase32Hex:          base32.HexEncoding.EncodeToString(data),
		EncodingBase32HexNoPadding: base32HexNoPadding.EncodeToString(data),
		EncodingBase64Std:          base64.StdEncoding.EncodeToString(data),
		EncodingBase64RawStd:       base64RawStd.EncodeToString(data),
		EncodingBase64URL:          base64.URLEncoding.EncodeToString(data),
		EncodingBase64RawURL:       base64RawURL.EncodeToString(data),
		EncodingBase91:             encodeBase91(data),
		EncodingHexUpper:           hexUpper(data),
	}

	asciiBuf := make([]byte, ascii85.MaxEncodedLen(len(data)))
	n := ascii85.Encode(asciiBuf, data)
	out[EncodingBase85ASCII] = string(asciiBuf[:n])

	return out, nil
}

// DecodeContent decodes the provided text using the given encoding key.
func DecodeContent(kind, input string) (string, error) {
	decoder, ok := encodingDecoders[kind]
	if !ok {
		return "", fmt.Errorf("unsupported decode type %s", kind)
	}
	data, err := decoder(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// HashContent returns the digests of the input across the standard library hash functions.
func HashContent(input string) map[string]string {
	data := []byte(input)
	out := map[string]string{}

	sumMD5 := md5.Sum(data)
	out["md5"] = hex.EncodeToString(sumMD5[:])

	sumSHA1 := sha1.Sum(data)
	out["sha1"] = hex.EncodeToString(sumSHA1[:])

	sumSHA224 := sha256.Sum224(data)
	out["sha224"] = hex.EncodeToString(sumSHA224[:])

	sumSHA256 := sha256.Sum256(data)
	out["sha256"] = hex.EncodeToString(sumSHA256[:])

	sumSHA384 := sha512.Sum384(data)
	out["sha384"] = hex.EncodeToString(sumSHA384[:])

	sumSHA512 := sha512.Sum512(data)
	out["sha512"] = hex.EncodeToString(sumSHA512[:])

	sumSHA512_224 := sha512.Sum512_224(data)
	out["sha512_224"] = hex.EncodeToString(sumSHA512_224[:])

	sumSHA512_256 := sha512.Sum512_256(data)
	out["sha512_256"] = hex.EncodeToString(sumSHA512_256[:])

	out["crc32_ieee"] = fmt.Sprintf("%08x", crc32.ChecksumIEEE(data))
	out["crc32_castagnoli"] = fmt.Sprintf("%08x", crc32.Checksum(data, crc32Castagnoli))
	out["crc64_iso"] = fmt.Sprintf("%016x", crc64.Checksum(data, crc64ISOTable))
	out["crc64_ecma"] = fmt.Sprintf("%016x", crc64.Checksum(data, crc64ECMATable))
	out["adler32"] = fmt.Sprintf("%08x", adler32.Checksum(data))

	out["fnv32"] = fmt.Sprintf("%08x", digest32(fnv.New32(), data))
	out["fnv32a"] = fmt.Sprintf("%08x", digest32(fnv.New32a(), data))
	out["fnv64"] = fmt.Sprintf("%016x", digest64(fnv.New64(), data))
	out["fnv64a"] = fmt.Sprintf("%016x", digest64(fnv.New64a(), data))
	out["fnv128"] = digestHash(fnv.New128(), data)
	out["fnv128a"] = digestHash(fnv.New128a(), data)

	return out
}

var encodingDecoders = map[string]func(string) ([]byte, error){
	EncodingBase32Std: func(s string) ([]byte, error) {
		return base32.StdEncoding.DecodeString(s)
	},
	EncodingBase32StdNoPadding: func(s string) ([]byte, error) {
		return base32StdNoPadding.DecodeString(s)
	},
	EncodingBase32Hex: func(s string) ([]byte, error) {
		return base32.HexEncoding.DecodeString(s)
	},
	EncodingBase32HexNoPadding: func(s string) ([]byte, error) {
		return base32HexNoPadding.DecodeString(s)
	},
	EncodingBase64Std: func(s string) ([]byte, error) {
		return base64.StdEncoding.DecodeString(s)
	},
	EncodingBase64RawStd: func(s string) ([]byte, error) {
		return base64RawStd.DecodeString(s)
	},
	EncodingBase64URL: func(s string) ([]byte, error) {
		return base64.URLEncoding.DecodeString(s)
	},
	EncodingBase64RawURL: func(s string) ([]byte, error) {
		return base64RawURL.DecodeString(s)
	},
	EncodingBase85ASCII: decodeBase85,
	EncodingBase91:      decodeBase91,
	EncodingHexUpper: func(s string) ([]byte, error) {
		return hex.DecodeString(strings.TrimSpace(s))
	},
}

func encodeBase91(data []byte) string {
	var out []byte
	var value uint
	var bits uint
	for _, b := range data {
		value |= uint(b) << bits
		bits += 8
		for bits > 13 {
			encoded := value & 8191
			if encoded > 88 {
				value >>= 13
				bits -= 13
			} else {
				encoded = value & 16383
				value >>= 14
				bits -= 14
			}
			out = append(out, base91Alphabet[encoded%91], base91Alphabet[encoded/91])
		}
	}
	if bits > 0 {
		out = append(out, base91Alphabet[value%91])
		if bits > 7 || value > 90 {
			out = append(out, base91Alphabet[value/91])
		}
	}
	return string(out)
}

func decodeBase85(input string) ([]byte, error) {
	reader := ascii85.NewDecoder(strings.NewReader(input))
	return io.ReadAll(reader)
}

func decodeBase91(input string) ([]byte, error) {
	var value uint
	var bits uint
	var b = -1
	var out []byte
	for i := 0; i < len(input); i++ {
		c := input[i]
		index := base91Lookup[c]
		if index == -1 {
			return nil, fmt.Errorf("invalid base91 character %q", c)
		}
		if b == -1 {
			b = index
			continue
		}
		b += index * 91
		value |= uint(b) << bits
		if (b & 8191) > 88 {
			bits += 13
		} else {
			bits += 14
		}
		for bits >= 8 {
			out = append(out, byte(value&255))
			value >>= 8
			bits -= 8
		}
		b = -1
	}
	if b != -1 {
		value |= uint(b) << bits
		out = append(out, byte(value&255))
	}
	return out, nil
}

func initBase91Lookup() [256]int {
	var table [256]int
	for i := range table {
		table[i] = -1
	}
	for idx, b := range base91Alphabet {
		table[b] = idx
	}
	return table
}

var base91Alphabet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&()*+,./:;?@[]^_`{|}~\"")

func hexUpper(data []byte) string {
	buf := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(buf, data)
	return strings.ToUpper(string(buf))
}

func digest32(h hash.Hash32, data []byte) uint32 {
	if len(data) == 0 {
		return h.Sum32()
	}
	_, _ = h.Write(data)
	return h.Sum32()
}

func digest64(h hash.Hash64, data []byte) uint64 {
	if len(data) == 0 {
		return h.Sum64()
	}
	_, _ = h.Write(data)
	return h.Sum64()
}

func digestHash(h hash.Hash, data []byte) string {
	if len(data) > 0 {
		_, _ = h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func URLEncode(input string) string {
	return url.QueryEscape(input)
}

func URLDecode(input string) (string, error) {
	return url.QueryUnescape(input)
}

type JWTParts struct {
	Header    string
	Payload   string
	Signature string
	Algorithm string
}

func JWTEncode(payloadInput, secret, algorithm string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("secret is required")
	}
	if algorithm == "" {
		algorithm = "HS256"
	}
	payloadBytes, err := compactJSON(payloadInput)
	if err != nil {
		return "", fmt.Errorf("payload must be valid JSON: %w", err)
	}
	header := map[string]string{
		"typ": "JWT",
		"alg": algorithm,
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := headerEncoded + "." + payloadEncoded
	signature, err := signJWT(signingInput, secret, algorithm)
	if err != nil {
		return "", err
	}
	return signingInput + "." + signature, nil
}

func JWTDecode(token string) (JWTParts, error) {
	var parts JWTParts
	token = strings.TrimSpace(token)
	segments := strings.Split(token, ".")
	if len(segments) < 2 {
		return parts, errors.New("invalid JWT token")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		return parts, fmt.Errorf("invalid header: %w", err)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return parts, fmt.Errorf("invalid payload: %w", err)
	}
	headerPretty, err := prettyJSON(headerJSON)
	if err != nil {
		return parts, fmt.Errorf("invalid header JSON: %w", err)
	}
	payloadPretty, err := prettyJSON(payloadJSON)
	if err != nil {
		return parts, fmt.Errorf("invalid payload JSON: %w", err)
	}
	parts.Header = headerPretty
	parts.Payload = payloadPretty
	if len(segments) > 2 {
		parts.Signature = segments[2]
	}
	var headerData map[string]any
	if err := json.Unmarshal(headerJSON, &headerData); err == nil {
		if alg, ok := headerData["alg"].(string); ok {
			parts.Algorithm = alg
		}
	}
	return parts, nil
}

func compactJSON(input string) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(input)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func prettyJSON(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func signJWT(signingInput, secret, algorithm string) (string, error) {
	var mac hash.Hash
	switch algorithm {
	case "HS256", "":
		mac = hmac.New(sha256.New, []byte(secret))
	case "HS384":
		mac = hmac.New(sha512.New384, []byte(secret))
	case "HS512":
		mac = hmac.New(sha512.New, []byte(secret))
	default:
		return "", fmt.Errorf("unsupported algorithm %s", algorithm)
	}
	_, _ = mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature), nil
}
