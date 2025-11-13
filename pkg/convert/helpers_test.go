package convert

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_exportName(t *testing.T) {
	require.Equal(t, "UserName", exportName("user_name"))
	require.Equal(t, "HTTPServerV2", exportName("HTTP_server_v2"))
	require.Equal(t, "A1B2", exportName("a1 b2"))
}

func Benchmark_exportName(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = exportName("user_name")
	}
}

func Fuzz_exportName(f *testing.F) {
	f.Add("user_name")
	f.Fuzz(func(t *testing.T, s string) {
		_ = exportName(s)
	})
}

func Test_lowerFirst(t *testing.T) {
	require.Equal(t, "userName", lowerFirst("UserName"))
	require.Equal(t, "httpServerV2", lowerFirst("HTTPServerV2"))
}

func Benchmark_lowerFirst(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lowerFirst("UserName")
	}
}

func Fuzz_lowerFirst(f *testing.F) {
	f.Add("UserName")
	f.Fuzz(func(t *testing.T, s string) {
		_ = lowerFirst(s)
	})
}

func Test_splitWords(t *testing.T) {
	parts := splitWords("HTTPServerV2Core")
	require.Equal(t, []string{"HTTP", "Server", "V2", "Core"}, parts)
}

func Benchmark_splitWords(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitWords("HTTPServerV2Core")
	}
}

func Fuzz_splitWords(f *testing.F) {
	f.Add("HelloWorld123")
	f.Fuzz(func(t *testing.T, s string) {
		_ = splitWords(s)
	})
}

func Test_isAllUpper(t *testing.T) {
	require.True(t, isAllUpper("HTTP"))
	require.False(t, isAllUpper("Http"))
}

func Benchmark_isAllUpper(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isAllUpper("HTTP")
	}
}

func Fuzz_isAllUpper(f *testing.F) {
	f.Add("ABC")
	f.Fuzz(func(t *testing.T, s string) {
		_ = isAllUpper(s)
	})
}

func Test_findMatchingBrace(t *testing.T) {
	s := "func x() { if true { return } } // end"
	idx := strings.Index(s, "{")
	require.GreaterOrEqual(t, idx, 0)
	j := findMatchingBrace(s, idx)
	require.NotEqual(t, -1, j)
	require.Greater(t, j, idx)
}

func Benchmark_findMatchingBrace(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	s := "a{b{c}d}e"
	idx := strings.Index(s, "{")
	for i := 0; i < b.N; i++ {
		_ = findMatchingBrace(s, idx)
	}
}

func Fuzz_findMatchingBrace(f *testing.F) {
	f.Add("a{b}c")
	f.Fuzz(func(t *testing.T, s string) {
		idx := strings.Index(s, "{")
		if idx < 0 { t.Skip() }
		_ = findMatchingBrace(s, idx)
	})
}
