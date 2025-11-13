package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseGoStructDefinitions(t *testing.T) {
	code := `package main
type A struct {
	ID int ` + "`json:\"id\"`" + `
	Name string
}`
	defs, err := parseGoStructDefinitions(code)
	require.NoError(t, err)
	require.NotEmpty(t, defs)
	require.Equal(t, "A", defs[0].Name)
	require.GreaterOrEqual(t, len(defs[0].Fields), 2)
	require.Equal(t, "id", defs[0].Fields[0].JSONName)
}

func Benchmark_parseGoStructDefinitions(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	code := `package main
type A struct { ID int; Name string }`
	for i := 0; i < b.N; i++ {
		_, _ = parseGoStructDefinitions(code)
	}
}

func Fuzz_parseGoStructDefinitions(f *testing.F) {
	f.Add("type X struct { N int }")
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = parseGoStructDefinitions(input)
	})
}
