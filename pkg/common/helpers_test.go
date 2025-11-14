package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportName(t *testing.T) {
	require.Equal(t, "UserName", ExportName("user_name"))
	require.Equal(t, "HTTPServerV2", ExportName("HTTP_server_v2"))
	require.Equal(t, "A1B2", ExportName("a1 b2"))
}

func BenchmarkExportName(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExportName("user_name")
	}
}

func FuzzExportName(f *testing.F) {
	f.Add("user_name")
	f.Fuzz(func(t *testing.T, s string) {
		_ = ExportName(s)
	})
}

func TestLowerFirst(t *testing.T) {
	require.Equal(t, "userName", LowerFirst("UserName"))
	require.Equal(t, "httpServerV2", LowerFirst("HTTPServerV2"))
}

func BenchmarkLowerFirst(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LowerFirst("UserName")
	}
}

func FuzzLowerFirst(f *testing.F) {
	f.Add("UserName")
	f.Fuzz(func(t *testing.T, s string) {
		_ = LowerFirst(s)
	})
}

func TestSplitWords(t *testing.T) {
	parts := SplitWords("HTTPServerV2Core")
	require.Equal(t, []string{"HTTP", "Server", "V2", "Core"}, parts)
}

func BenchmarkSplitWords(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitWords("HTTPServerV2Core")
	}
}

func FuzzSplitWords(f *testing.F) {
	f.Add("HelloWorld123")
	f.Fuzz(func(t *testing.T, s string) {
		_ = SplitWords(s)
	})
}

func TestIsAllUpper(t *testing.T) {
	require.True(t, IsAllUpper("HTTP"))
	require.False(t, IsAllUpper("Http"))
}

func BenchmarkIsAllUpper(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsAllUpper("HTTP")
	}
}

func FuzzIsAllUpper(f *testing.F) {
	f.Add("ABC")
	f.Fuzz(func(t *testing.T, s string) {
		_ = IsAllUpper(s)
	})
}

func TestFindMatchingBrace(t *testing.T) {
	s := "func x() { if true { return } } // end"
	idx := strings.Index(s, "{")
	require.GreaterOrEqual(t, idx, 0)
	j := FindMatchingBrace(s, idx)
	require.NotEqual(t, -1, j)
	require.Greater(t, j, idx)
}

func BenchmarkFindMatchingBrace(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	s := "a{b{c}d}e"
	idx := strings.Index(s, "{")
	for i := 0; i < b.N; i++ {
		_ = FindMatchingBrace(s, idx)
	}
}

func FuzzFindMatchingBrace(f *testing.F) {
	f.Add("a{b}c")
	f.Fuzz(func(t *testing.T, s string) {
		idx := strings.Index(s, "{")
		if idx < 0 {
			t.Skip()
		}
		_ = FindMatchingBrace(s, idx)
	})
}
