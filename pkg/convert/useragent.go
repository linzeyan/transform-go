package convert

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
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

var (
	browserSources = map[string]string{
		"chrome":  "https://www.whatismybrowser.com/guides/the-latest-version/chrome",
		"firefox": "https://www.whatismybrowser.com/guides/the-latest-version/firefox",
		"opera":   "https://www.whatismybrowser.com/guides/the-latest-version/opera",
		"safari":  "https://www.whatismybrowser.com/guides/the-latest-version/safari",
		"edge":    "https://www.whatismybrowser.com/guides/the-latest-version/edge",
		"vivaldi": "https://www.whatismybrowser.com/guides/the-latest-version/vivaldi",
		"yandex":  "https://www.whatismybrowser.com/guides/the-latest-version/yandex-browser",
	}

	platformSources = map[string]string{
		"chrome-os": "https://www.whatismybrowser.com/guides/the-latest-version/chrome-os",
		"macos":     "https://www.whatismybrowser.com/guides/the-latest-version/macos",
		"ios":       "https://www.whatismybrowser.com/guides/the-latest-version/ios",
		"android":   "https://www.whatismybrowser.com/guides/the-latest-version/android",
		"windows":   "https://www.whatismybrowser.com/guides/the-latest-version/windows",
	}

	browserNames = map[string]string{
		"chrome":  "Chrome",
		"firefox": "Firefox",
		"opera":   "Opera",
		"safari":  "Safari",
		"edge":    "Edge",
		"vivaldi": "Vivaldi",
		"yandex":  "Yandex Browser",
	}

	platformNames = map[string]string{
		"chrome-os": "ChromeOS",
		"macos":     "macOS",
		"ios":       "iOS",
		"android":   "Android",
		"windows":   "Windows",
	}
)

var (
	latestDataMu    sync.RWMutex
	latestData      *versionCache
	fetchInProgress bool
	networkAllowed  = runtime.GOOS != "js" && runtime.GOARCH != "wasm"
)

const cacheTTL = 6 * time.Hour

var (
	restyClient   = resty.New().SetTimeout(20 * time.Second)
	fetchDocument = fetchDocumentHTTP
)

type versionCache struct {
	browsers  map[string][]tableRow
	platforms map[string][]tableRow
	fetchedAt time.Time
}

type tableRow map[string]string

type platformDetail struct {
	Name         string
	Version      string
	VersionLabel string
	Token        string
}

func init() {
	latestData = fallbackVersionCache()
}

// GenerateUserAgents fetches the latest browser + platform data and builds
// example user-agent strings. browser/os filters may be empty to list all.
func GenerateUserAgents(browser, os string) ([]UserAgentInfo, error) {
	cache, err := ensureLatestData(context.Background())
	if err != nil {
		return nil, err
	}

	browserFilter := normalizeBrowser(browser)
	platformFilter := normalizePlatform(os)

	results := buildUserAgents(cache, browserFilter, platformFilter)
	if len(results) == 0 {
		if browserFilter != "" || platformFilter != "" {
			return nil, fmt.Errorf("no user agents available for browser=%q platform=%q", browser, os)
		}
		return nil, errors.New("no user agent data available")
	}
	if len(results) > 10 {
		results = results[:10]
	}
	return results, nil
}

func ensureLatestData(ctx context.Context) (*versionCache, error) {
	latestDataMu.RLock()
	data := latestData
	expired := time.Since(data.fetchedAt) >= cacheTTL
	latestDataMu.RUnlock()

	if expired && networkAllowed {
		triggerBackgroundFetch()
	}
	return data, nil
}

func triggerBackgroundFetch() {
	if !networkAllowed {
		return
	}
	latestDataMu.Lock()
	if fetchInProgress {
		latestDataMu.Unlock()
		return
	}
	fetchInProgress = true
	latestDataMu.Unlock()

	go func() {
		defer func() {
			latestDataMu.Lock()
			fetchInProgress = false
			latestDataMu.Unlock()
		}()
		cache, err := fetchLatestData(context.Background())
		if err != nil {
			return
		}
		latestDataMu.Lock()
		latestData = cache
		latestDataMu.Unlock()
	}()
}

func fetchLatestData(ctx context.Context) (*versionCache, error) {
	cache := &versionCache{
		browsers:  make(map[string][]tableRow, len(browserSources)),
		platforms: make(map[string][]tableRow, len(platformSources)),
		fetchedAt: time.Now(),
	}

	for slug, url := range browserSources {
		doc, rows, err := fetchDocumentRows(ctx, url)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			if slug == "vivaldi" {
				rows = parseVivaldiDoc(doc)
			}
			if len(rows) == 0 {
				return nil, fmt.Errorf("no table data for %s", slug)
			}
		}
		cache.browsers[slug] = rows
	}

	for slug, url := range platformSources {
		_, rows, err := fetchDocumentRows(ctx, url)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("no table data for %s", slug)
		}
		cache.platforms[slug] = rows
	}

	return cache, nil
}

func fetchDocumentRows(ctx context.Context, url string) (*goquery.Document, []tableRow, error) {
	doc, err := fetchDocument(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	rows := extractLatestTable(doc)
	return doc, rows, nil
}

func fetchDocumentHTTP(ctx context.Context, url string) (*goquery.Document, error) {
	resp, err := restyClient.R().
		SetContext(ctx).
		SetHeader("User-Agent", "Mozilla/5.0 (compatible; transform-go/1.0; +https://github.com/linzeyan/transform-go)").
		Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode())
	}
	return goquery.NewDocumentFromReader(bytes.NewReader(resp.Body()))
}

func extractLatestTable(doc *goquery.Document) []tableRow {
	var rows []tableRow
	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		headerCells := table.Find("thead th")
		if headerCells.Length() == 0 {
			headerCells = table.Find("tr").First().Find("th")
		}
		if headerCells.Length() == 0 {
			headerCells = table.Find("tr").First().Find("td")
		}
		headers := make([]string, headerCells.Length())
		headerCells.Each(func(i int, sel *goquery.Selection) {
			headers[i] = strings.TrimSpace(sel.Text())
		})
		if len(headers) == 0 {
			return true
		}
		table.Find("tbody tr").Each(func(_ int, rowSel *goquery.Selection) {
			values := make(tableRow)
			rowSel.Find("td").Each(func(i int, cell *goquery.Selection) {
				if i < len(headers) {
					values[headers[i]] = strings.TrimSpace(cell.Text())
				}
			})
			if len(values) > 0 {
				rows = append(rows, values)
			}
		})
		if len(rows) == 0 {
			table.Find("tr").Each(func(idx int, rowSel *goquery.Selection) {
				if idx == 0 {
					return
				}
				values := make(tableRow)
				rowSel.Find("td").Each(func(i int, cell *goquery.Selection) {
					if i < len(headers) {
						values[headers[i]] = strings.TrimSpace(cell.Text())
					}
				})
				if len(values) > 0 {
					rows = append(rows, values)
				}
			})
		}
		return len(rows) == 0
	})
	return rows
}

func parseVivaldiDoc(doc *goquery.Document) []tableRow {
	var version, date string
	doc.Find("h2").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		text := strings.TrimSpace(sel.Text())
		if strings.HasPrefix(text, "The latest version of Vivaldi is") {
			parts := strings.Split(text, ":")
			if len(parts) == 2 {
				version = strings.TrimSpace(parts[1])
			}
			return false
		}
		return true
	})
	if version == "" {
		return nil
	}
	doc.Find("p").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		text := strings.TrimSpace(sel.Text())
		if strings.HasPrefix(text, "It was released") {
			date = strings.TrimSpace(strings.TrimPrefix(text, "It was released"))
			date = strings.Trim(date, ". ")
			return false
		}
		return true
	})
	return []tableRow{{"Platform": "Vivaldi", "Version": version, "Release Date": date}}
}

func normalizeBrowser(input string) string {
	slug := strings.ToLower(strings.TrimSpace(input))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	if _, ok := browserSources[slug]; ok {
		return slug
	}
	return ""
}

func normalizePlatform(input string) string {
	slug := strings.ToLower(strings.TrimSpace(input))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	if _, ok := platformSources[slug]; ok {
		return slug
	}
	return ""
}

func buildUserAgents(cache *versionCache, browserFilter, platformFilter string) []UserAgentInfo {
	orderedBrowsers := make([]string, 0, len(browserSources))
	for slug := range browserSources {
		orderedBrowsers = append(orderedBrowsers, slug)
	}
	sort.Strings(orderedBrowsers)

	var result []UserAgentInfo
	for _, browser := range orderedBrowsers {
		if browserFilter != "" && browser != browserFilter {
			continue
		}
		builder, ok := browserBuilders[browser]
		if !ok {
			continue
		}
		entries := builder(cache, platformFilter)
		result = append(result, entries...)
		if len(result) >= 10 {
			break
		}
	}
	return result
}

var browserBuilders = map[string]func(*versionCache, string) []UserAgentInfo{
	"chrome":  buildChromeUA,
	"edge":    buildEdgeUA,
	"firefox": buildFirefoxUA,
	"opera":   buildOperaUA,
	"safari":  buildSafariUA,
	"vivaldi": buildVivaldiUA,
	"yandex":  buildYandexUA,
}

func buildChromeUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	combos := []string{"windows", "macos", "android", "ios", "chrome-os"}
	return buildBlinkUA("Chrome", "chrome", combos, cache, platformFilter, func(version string, platform string, detail platformDetail) string {
		switch platform {
		case "windows":
			return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
		case "macos":
			return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", detail.Token, version)
		case "android":
			return fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36", detail.Version, version)
		case "ios":
			return fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/%s Mobile/15E148 Safari/604.1", detail.Token, version)
		case "chrome-os":
			return fmt.Sprintf("Mozilla/5.0 (X11; CrOS x86_64 %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", detail.Version, version)
		default:
			return ""
		}
	})
}

func buildEdgeUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	combos := []string{"windows", "macos", "android", "ios"}
	return buildBlinkUA("Edge", "edge", combos, cache, platformFilter, func(version, platform string, detail platformDetail) string {
		switch platform {
		case "windows":
			return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s", version, version)
		case "macos":
			return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15 Edg/%s", detail.Token, version)
		case "android":
			return fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36 EdgA/%s", detail.Version, version, version)
		case "ios":
			return fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 EdgiOS/%s", detail.Token, version)
		default:
			return ""
		}
	})
}

func buildOperaUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	combos := []string{"windows", "macos", "android"}
	return buildBlinkUA("Opera", "opera", combos, cache, platformFilter, func(version, platform string, detail platformDetail) string {
		chromeVer := version
		switch platform {
		case "windows":
			return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 OPR/%s", chromeVer, version)
		case "macos":
			return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 OPR/%s", detail.Token, chromeVer, version)
		case "android":
			return fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36 OPR/%s", detail.Version, chromeVer, version)
		default:
			return ""
		}
	})
}

func buildVivaldiUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	combos := []string{"windows", "macos"}
	return buildBlinkUA("Vivaldi", "vivaldi", combos, cache, platformFilter, func(version, platform string, detail platformDetail) string {
		chromeVer := version
		switch platform {
		case "windows":
			return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Vivaldi/%s", chromeVer, version)
		case "macos":
			return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Vivaldi/%s", detail.Token, chromeVer, version)
		default:
			return ""
		}
	})
}

func buildYandexUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	combos := []string{"windows", "macos", "android", "ios"}
	return buildBlinkUA("Yandex Browser", "yandex", combos, cache, platformFilter, func(version, platform string, detail platformDetail) string {
		switch platform {
		case "windows":
			return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 YaBrowser/%s", version, version)
		case "macos":
			return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 YaBrowser/%s", detail.Token, version, version)
		case "android":
			return fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36 YaApp_Android/%s YaSearchBrowser/%s", detail.Version, version, version, version)
		case "ios":
			return fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 YaBrowser/%s", detail.Token, version)
		default:
			return ""
		}
	})
}

func buildSafariUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	rows := cache.browsers["safari"]
	combos := []string{"macos", "ios"}
	var result []UserAgentInfo
	for _, platform := range combos {
		if platformFilter != "" && platform != platformFilter {
			continue
		}
		version := matchBrowserVersion(rows, platform)
		if version == "" {
			continue
		}
		detail := platformDetails(platform, cache.platforms[platform])
		var ua string
		if platform == "macos" {
			ua = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15", detail.Token, version)
		} else {
			ua = fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Mobile/15E148 Safari/604.1", detail.Token, version)
		}
		result = append(result, UserAgentInfo{
			UserAgent:      ua,
			BrowserName:    "Safari",
			BrowserVersion: version,
			OSName:         detail.Name,
			OSVersion:      detail.VersionLabel,
			EngineName:     "WebKit",
			EngineVersion:  "605.1.15",
		})
	}
	return result
}

func buildFirefoxUA(cache *versionCache, platformFilter string) []UserAgentInfo {
	rows := cache.browsers["firefox"]
	versionDesktop := findRowValue(rows, "Release Edition", "Firefox Standard Release", "Version")
	versioniOS := findRowValue(rows, "Release Edition", "Firefox iOS", "Version")
	versionAndroid := findRowValue(rows, "Release Edition", "Firefox Android", "Version")

	var result []UserAgentInfo
	detailWin := platformDetails("windows", cache.platforms["windows"])
	detailMac := platformDetails("macos", cache.platforms["macos"])
	detailAndroid := platformDetails("android", cache.platforms["android"])
	detailIOS := platformDetails("ios", cache.platforms["ios"])

	if versionDesktop != "" && (platformFilter == "" || platformFilter == "windows") {
		result = append(result, UserAgentInfo{
			UserAgent:      fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:%s) Gecko/20100101 Firefox/%s", versionDesktop, versionDesktop),
			BrowserName:    "Firefox",
			BrowserVersion: versionDesktop,
			OSName:         detailWin.Name,
			OSVersion:      detailWin.VersionLabel,
			EngineName:     "Gecko",
			EngineVersion:  versionDesktop,
		})
	}
	if versionDesktop != "" && (platformFilter == "" || platformFilter == "macos") {
		result = append(result, UserAgentInfo{
			UserAgent:      fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s; rv:%s) Gecko/20100101 Firefox/%s", detailMac.Token, versionDesktop, versionDesktop),
			BrowserName:    "Firefox",
			BrowserVersion: versionDesktop,
			OSName:         detailMac.Name,
			OSVersion:      detailMac.VersionLabel,
			EngineName:     "Gecko",
			EngineVersion:  versionDesktop,
		})
	}
	if versionAndroid != "" && (platformFilter == "" || platformFilter == "android") {
		result = append(result, UserAgentInfo{
			UserAgent:      fmt.Sprintf("Mozilla/5.0 (Android %s; Mobile; rv:%s) Gecko/%s Firefox/%s", detailAndroid.Version, versionAndroid, versionAndroid, versionAndroid),
			BrowserName:    "Firefox",
			BrowserVersion: versionAndroid,
			OSName:         detailAndroid.Name,
			OSVersion:      detailAndroid.VersionLabel,
			EngineName:     "Gecko",
			EngineVersion:  versionAndroid,
		})
	}
	if versioniOS != "" && (platformFilter == "" || platformFilter == "ios") {
		result = append(result, UserAgentInfo{
			UserAgent:      fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/%s Mobile/15E148 Safari/605.1.15", detailIOS.Token, versioniOS),
			BrowserName:    "Firefox",
			BrowserVersion: versioniOS,
			OSName:         detailIOS.Name,
			OSVersion:      detailIOS.VersionLabel,
			EngineName:     "Gecko",
			EngineVersion:  versioniOS,
		})
	}
	return result
}

func buildBlinkUA(browserName, slug string, combos []string, cache *versionCache, platformFilter string, template func(version, platform string, detail platformDetail) string) []UserAgentInfo {
	rows := cache.browsers[slug]
	var result []UserAgentInfo
	for _, platform := range combos {
		if platformFilter != "" && platform != platformFilter {
			continue
		}
		version := matchBrowserVersion(rows, platform)
		if version == "" {
			continue
		}
		detail := platformDetails(platform, cache.platforms[platform])
		ua := template(version, platform, detail)
		if ua == "" {
			continue
		}
		result = append(result, UserAgentInfo{
			UserAgent:      ua,
			BrowserName:    browserName,
			BrowserVersion: version,
			OSName:         detail.Name,
			OSVersion:      detail.VersionLabel,
			EngineName:     "Blink",
			EngineVersion:  version,
		})
	}
	return result
}

func platformDetails(slug string, rows []tableRow) platformDetail {
	detail := platformDetail{Name: platformNames[slug]}
	if len(rows) == 0 {
		detail.Token = "10_0"
		return detail
	}
	row := rows[0]
	switch slug {
	case "windows":
		version := row["Version Number"]
		build := row["Build"]
		detail.Version = build
		switch {
		case version != "" && build != "":
			detail.VersionLabel = fmt.Sprintf("%s (%s)", version, build)
		case version != "":
			detail.VersionLabel = version
		default:
			detail.VersionLabel = build
		}
		detail.Token = "10.0"
	case "macos":
		version := row["Version Number"]
		detail.Version = version
		detail.VersionLabel = version
		detail.Token = strings.ReplaceAll(version, ".", "_")
	case "ios":
		version := row["Version"]
		detail.Version = version
		detail.VersionLabel = version
		detail.Token = strings.ReplaceAll(version, ".", "_")
	case "android":
		version := row["Version Number"]
		detail.Version = version
		detail.VersionLabel = version
		detail.Token = strings.ReplaceAll(version, ".", "_")
	case "chrome-os":
		version := row["Version"]
		if version == "" {
			version = row["Platform Version"]
		}
		detail.Version = version
		detail.VersionLabel = version
		detail.Token = version
	default:
		detail.VersionLabel = row["Version"]
		detail.Token = detail.VersionLabel
	}
	return detail
}

func matchBrowserVersion(rows []tableRow, platform string) string {
	platformName := platformNames[platform]
	for _, row := range rows {
		for _, value := range row {
			if strings.Contains(strings.ToLower(value), strings.ToLower(platformName)) {
				if version := firstNonEmpty(row, "Version", "Version Number"); version != "" {
					return version
				}
			}
		}
	}
	if len(rows) > 0 {
		return firstNonEmpty(rows[0], "Version", "Version Number")
	}
	return ""
}

func findRowValue(rows []tableRow, key, contains, valueKey string) string {
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row[key]), strings.ToLower(contains)) {
			if v := row[valueKey]; v != "" {
				return v
			}
		}
	}
	return ""
}

func firstNonEmpty(row tableRow, keys ...string) string {
	for _, key := range keys {
		if v := row[key]; v != "" {
			return v
		}
	}
	return ""
}

func fallbackVersionCache() *versionCache {
	return &versionCache{
		browsers:  cloneTableData(defaultBrowserData),
		platforms: cloneTableData(defaultPlatformData),
		fetchedAt: time.Unix(0, 0),
	}
}

func cloneTableData(src map[string][]tableRow) map[string][]tableRow {
	dst := make(map[string][]tableRow, len(src))
	for key, rows := range src {
		copied := make([]tableRow, len(rows))
		for i, row := range rows {
			clone := make(tableRow, len(row))
			for k, v := range row {
				clone[k] = v
			}
			copied[i] = clone
		}
		dst[key] = copied
	}
	return dst
}

var defaultBrowserData = map[string][]tableRow{
	"chrome": {
		{"Platform": "Chrome on Windows", "Version": "142.0.7444.136", "Release Date": "2025-11-11"},
		{"Platform": "Chrome on macOS", "Version": "142.0.7444.134", "Release Date": "2025-11-11"},
		{"Platform": "Chrome on Linux", "Version": "142.0.7444.162", "Release Date": "2025-11-11"},
		{"Platform": "Chrome on Android", "Version": "142.0.7444.139", "Release Date": "2025-11-11"},
		{"Platform": "Chrome on iOS", "Version": "142.0.7444.148", "Release Date": "2025-11-11"},
	},
	"firefox": {
		{"Release Edition": "Firefox Standard Release", "Platform": "Desktop", "Version": "145.0", "Release Date": "2025-11-11"},
		{"Release Edition": "Firefox Extended Support Release", "Platform": "Desktop", "Version": "140.5.0", "Release Date": "2025-11-11"},
		{"Release Edition": "Firefox iOS", "Platform": "Mobile", "Version": "145.0", "Release Date": "2025-11-11"},
		{"Release Edition": "Firefox Android", "Platform": "Mobile", "Version": "145.0", "Release Date": "2025-11-11"},
	},
	"opera": {
		{"Platform": "Opera on Desktop", "Version": "123.0.5669.18", "Release Date": "2025-11-06"},
		{"Platform": "Opera on Android", "Version": "76.2.4027.73374", "Release Date": "2023-10-08"},
	},
	"safari": {
		{"Platform": "Safari on macOS (Laptops and Desktops)", "Version": "26.0", "Release Date": "2025-09-15"},
		{"Platform": "Safari on iOS (iPhone, iPad and iPod)", "Version": "26.0", "Release Date": "2025-09-15"},
	},
	"edge": {
		{"Platform": "Edge on Windows", "Version": "142.0.3595.80", "Release Date": "2025-11-13"},
		{"Platform": "Edge on macOS", "Version": "142.0.3595.80", "Release Date": "2025-11-13"},
		{"Platform": "Edge on Linux", "Version": "142.0.3595.80", "Release Date": "2025-11-13"},
		{"Platform": "Edge on iOS", "Version": "142.3595.66", "Release Date": "2025-11-11"},
		{"Platform": "Edge on Android", "Version": "142.0.3595.66", "Release Date": "2025-11-13"},
	},
	"vivaldi": {
		{"Platform": "Vivaldi", "Version": "7.6.3797.63", "Release Date": "2025-10-09"},
	},
	"yandex": {
		{"Platform": "Yandex Browser on Windows", "Version": "25.10.0.2516", "Release Date": "2025-11-11"},
		{"Platform": "Yandex Browser on macOS", "Version": "25.10.0.2516", "Release Date": "2025-11-11"},
		{"Platform": "Yandex Browser on iOS", "Version": "25.10.5.774", "Release Date": "2025-11-13"},
		{"Platform": "Yandex Browser on Android", "Version": "25.10.3.136", "Release Date": "2025-11-13"},
	},
}

var defaultPlatformData = map[string][]tableRow{
	"windows": {
		{"Platform": "Windows 11", "Version Number": "25H2", "Build": "26200.7171", "Release Date": "2025-11-11"},
	},
	"macos": {
		{"Platform": "macOS", "Version Number": "15.7.2", "Release Date": "2025-11-04"},
	},
	"ios": {
		{"Platform": "iOS on iPhone, iPad & iPod", "Version": "18.7.2", "Release Date": "2025-11-06"},
	},
	"android": {
		{"Platform": "Android (Standard)", "Version Number": "16.0", "Release Date": "2025-06-10"},
	},
	"chrome-os": {
		{"Platform": "ChromeOS on Chromebooks", "Platform Version": "16181.61.0", "Version": "134.0.6998.198", "Release Date": "2025-04-11"},
	},
}
