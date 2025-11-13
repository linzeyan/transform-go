package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_FormatContent_JSON(t *testing.T) {
	const input = `{"name":"Ricky","age":27}`
	pretty, err := FormatContent("JSON", input, false)
	require.NoError(t, err)
	require.Contains(t, pretty, "\n")
	require.Contains(t, pretty, `"name": "Ricky"`)

	minified, err := FormatContent("JSON", pretty, true)
	require.NoError(t, err)
	require.Equal(t, `{"age":27,"name":"Ricky"}`, minified)
}

func Test_FormatContent_GoStruct(t *testing.T) {
	const src = "type demo struct{ID string}"
	out, err := FormatContent("Go Struct", src, false)
	require.NoError(t, err)
	require.Contains(t, out, "type demo struct")
	require.Contains(t, out, "ID string")

	minified, err := FormatContent("Go Struct", out, true)
	require.NoError(t, err)
	require.Equal(t, out, minified)
}

func Test_FormatContent_GraphQL(t *testing.T) {
	const src = "type A {name:String age:Int}"
	formatted, err := FormatContent("GraphQL Schema", src, false)
	require.NoError(t, err)
	require.Contains(t, formatted, "type A")
	require.Contains(t, formatted, "name: String")
}

func Test_ConvertFormats_SpecialCases(t *testing.T) {
	out, err := ConvertFormats("Go Struct", "GraphQL Schema", sampleGoStruct)
	require.NoError(t, err)
	require.Contains(t, out, "type User")

	back, err := ConvertFormats("GraphQL Schema", "Go Struct", out)
	require.NoError(t, err)
	require.Contains(t, back, "type User struct")

	proto, err := ConvertFormats("Go Struct", "Protobuf", sampleGoStruct)
	require.NoError(t, err)
	require.Contains(t, proto, "message User")

	goStruct, err := ConvertFormats("Protobuf", "Go Struct", proto)
	require.NoError(t, err)
	require.Contains(t, goStruct, "type User struct")
}

func Test_FormatContent_InvalidFormat(t *testing.T) {
	_, err := FormatContent("Unknown", "data", false)
	require.Error(t, err)
}
