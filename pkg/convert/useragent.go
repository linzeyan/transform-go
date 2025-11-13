package convert

import (
	"fmt"
	"strings"
)

// UserAgentInfo represents a generated user agent entry.
type UserAgentInfo struct {
	UserAgent      string `json:"userAgent"`
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
	OSName         string `json:"osName"`
	OSVersion      string `json:"osVersion"`
	EngineName     string `json:"engineName"`
	EngineVersion  string `json:"engineVersion"`
}

var userAgentCatalog = []UserAgentInfo{
	{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.160 Safari/537.36",
		BrowserName:    "Chrome",
		BrowserVersion: "121.0.6167.160",
		OSName:         "Windows",
		OSVersion:      "10",
		EngineName:     "Blink",
		EngineVersion:  "121.0.6167.160",
	},
	{
		UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.85 Safari/537.36",
		BrowserName:    "Chrome",
		BrowserVersion: "121.0.6167.85",
		OSName:         "macOS",
		OSVersion:      "13.6.5",
		EngineName:     "Blink",
		EngineVersion:  "121.0.6167.85",
	},
	{
		UserAgent:      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.79 Safari/537.36",
		BrowserName:    "Chrome",
		BrowserVersion: "121.0.6167.79",
		OSName:         "Linux",
		OSVersion:      "x86_64",
		EngineName:     "Blink",
		EngineVersion:  "121.0.6167.79",
	},
	{
		UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
		BrowserName:    "Safari",
		BrowserVersion: "17.2",
		OSName:         "macOS",
		OSVersion:      "14.2",
		EngineName:     "WebKit",
		EngineVersion:  "605.1.15",
	},
	{
		UserAgent:      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
		BrowserName:    "Safari",
		BrowserVersion: "17.2",
		OSName:         "iOS",
		OSVersion:      "17.2",
		EngineName:     "WebKit",
		EngineVersion:  "605.1.15",
	},
	{
		UserAgent:      "Mozilla/5.0 (iPad; CPU OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
		BrowserName:    "Safari",
		BrowserVersion: "17.2",
		OSName:         "iPadOS",
		OSVersion:      "17.2",
		EngineName:     "WebKit",
		EngineVersion:  "605.1.15",
	},
	{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
		BrowserName:    "Firefox",
		BrowserVersion: "123.0",
		OSName:         "Windows",
		OSVersion:      "10",
		EngineName:     "Gecko",
		EngineVersion:  "123.0",
	},
	{
		UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.2; rv:123.0) Gecko/20100101 Firefox/123.0",
		BrowserName:    "Firefox",
		BrowserVersion: "123.0",
		OSName:         "macOS",
		OSVersion:      "14.2",
		EngineName:     "Gecko",
		EngineVersion:  "123.0",
	},
	{
		UserAgent:      "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:123.0) Gecko/20100101 Firefox/123.0",
		BrowserName:    "Firefox",
		BrowserVersion: "123.0",
		OSName:         "Linux",
		OSVersion:      "Ubuntu",
		EngineName:     "Gecko",
		EngineVersion:  "123.0",
	},
	{
		UserAgent:      "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.165 Mobile Safari/537.36",
		BrowserName:    "Chrome",
		BrowserVersion: "121.0.6167.165",
		OSName:         "Android",
		OSVersion:      "14",
		EngineName:     "Blink",
		EngineVersion:  "121.0.6167.165",
	},
	{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edg/121.0.2277.83 Safari/537.36",
		BrowserName:    "Edge",
		BrowserVersion: "121.0.2277.83",
		OSName:         "Windows",
		OSVersion:      "10",
		EngineName:     "Blink",
		EngineVersion:  "121.0.2277.83",
	},
	{
		UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36 (KHTML, like Gecko) Edg/121.0.2277.62 Safari/537.36",
		BrowserName:    "Edge",
		BrowserVersion: "121.0.2277.62",
		OSName:         "macOS",
		OSVersion:      "14.2",
		EngineName:     "Blink",
		EngineVersion:  "121.0.2277.62",
	},
	{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Brave/1.62.162 Chrome/121.0.6167.140 Safari/537.36",
		BrowserName:    "Brave",
		BrowserVersion: "1.62.162",
		OSName:         "Windows",
		OSVersion:      "10",
		EngineName:     "Blink",
		EngineVersion:  "121.0.6167.140",
	},
	{
		UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36 (KHTML, like Gecko) Brave/1.62.153 Chrome/121.0.6167.110 Safari/537.36",
		BrowserName:    "Brave",
		BrowserVersion: "1.62.153",
		OSName:         "macOS",
		OSVersion:      "14.2",
		EngineName:     "Blink",
		EngineVersion:  "121.0.6167.110",
	},
}

// GenerateUserAgents returns up to 10 user-agent strings filtered by browser and OS.
func GenerateUserAgents(browser, os string) ([]UserAgentInfo, error) {
	results := make([]UserAgentInfo, 0, 10)
	matchBrowser := strings.TrimSpace(strings.ToLower(browser))
	matchOS := strings.TrimSpace(strings.ToLower(os))
	for _, entry := range userAgentCatalog {
		if matchBrowser != "" && !strings.EqualFold(entry.BrowserName, matchBrowser) {
			continue
		}
		if matchOS != "" && !strings.EqualFold(entry.OSName, matchOS) {
			continue
		}
		results = append(results, entry)
		if len(results) == 10 {
			break
		}
	}
	if len(results) == 0 {
		limit := 10
		if len(userAgentCatalog) < 10 {
			limit = len(userAgentCatalog)
		}
		results = append(results, userAgentCatalog[:limit]...)
	}
	return results, nil
}

// DescribeUserAgent creates a short human-readable summary.
func DescribeUserAgent(info UserAgentInfo) string {
	return fmt.Sprintf("%s %s · %s %s · %s %s",
		info.BrowserName, info.BrowserVersion,
		info.OSName, info.OSVersion,
		info.EngineName, info.EngineVersion,
	)
}
