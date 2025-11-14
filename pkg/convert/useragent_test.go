package convert

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

func TestGenerateUserAgents(t *testing.T) {
	origFetch := fetchDocument
	origBrowserSources := browserSources
	origPlatformSources := platformSources
	origBrowserNames := browserNames
	origPlatformNames := platformNames
	origClient := restyClient
	defer func() {
		fetchDocument = origFetch
		browserSources = origBrowserSources
		platformSources = origPlatformSources
		browserNames = origBrowserNames
		platformNames = origPlatformNames
		restyClient = origClient
		latestDataMu.Lock()
		latestData = fallbackVersionCache()
		fetchInProgress = false
		latestDataMu.Unlock()
	}()

	browserSources = map[string]string{
		"chrome": "chrome-test",
	}
	platformSources = map[string]string{
		"windows": "windows-test",
	}
	browserNames = map[string]string{"chrome": "Chrome"}
	platformNames = map[string]string{"windows": "Windows"}

	samplePages := map[string]string{
		"chrome-test":  `<html><body><table><thead><tr><th>Platform</th><th>Version</th><th>Release Date</th></tr></thead><tbody><tr><td>Chrome on Windows</td><td>123.0.0.1</td><td>2025-11-01</td></tr></tbody></table></body></html>`,
		"windows-test": `<html><body><table><thead><tr><th>Platform</th><th>Version Number</th><th>Build</th><th>Release Date</th></tr></thead><tbody><tr><td>Windows 11</td><td>25H2</td><td>26200.111</td><td>2025-11-01</td></tr></tbody></table></body></html>`,
	}

	restyClient = nil
	fetchDocument = func(_ context.Context, url string) (*goquery.Document, error) {
		html := samplePages[url]
		return goquery.NewDocumentFromReader(strings.NewReader(html))
	}

	cache, err := fetchLatestData(context.Background())
	require.NoError(t, err)
	cache.fetchedAt = time.Now()
	latestDataMu.Lock()
	latestData = cache
	fetchInProgress = false
	latestDataMu.Unlock()

	list, err := GenerateUserAgents("chrome", "windows")
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, entry := range list {
		require.Equal(t, "Chrome", entry.BrowserName)
		require.Contains(t, entry.UserAgent, "Chrome/123.0.0.1")
	}
}
