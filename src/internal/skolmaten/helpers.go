package skolmaten

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultAPIBase  = "https://skolmaten.se/api/4"
	defaultSiteBase = "https://skolmaten.se"
	defaultTimeout  = 20 * time.Second
)

var (
	indexScriptRE = regexp.MustCompile(`(?i)src=["']([^"']*/assets/index-[^"']+\.js)["']`)
	viteTokenRE   = regexp.MustCompile(`VITE_CLIENT_TOKEN:"([^"]+)"`)
	headerTokenRE = regexp.MustCompile(`"Client-Token"\s*:\s*"([^"]+)"`)
)

// Client fetches school lunch menus from skolmaten.se.
type Client struct {
	hc        *http.Client
	token     string
	refreshed bool // true after a successful SPA token discovery this run
	apiBase   string
	siteBase  string
}

// NewClient returns a skolmaten API client. hc may be nil (20s timeout client).
// token is sent as the Client-Token header.
func NewClient(hc *http.Client, token string) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		hc:       hc,
		token:    strings.TrimSpace(token),
		apiBase:  defaultAPIBase,
		siteBase: defaultSiteBase,
	}
}

// TokenRefreshed reports whether this client replaced its token via SPA discovery.
func (c *Client) TokenRefreshed() bool {
	return c != nil && c.refreshed
}

// ISOWeeksCovering returns numWeeks consecutive ISO weeks starting at the ISO week of from
// (in loc, typically Europe/Stockholm). DIGEST_WEEKS=1 → only the current week.
func ISOWeeksCovering(from time.Time, numWeeks int, loc *time.Location) []YearWeek {
	if loc == nil {
		loc = time.UTC
	}
	if numWeeks < 1 {
		numWeeks = 1
	}
	y, w := from.In(loc).ISOWeek()
	out := make([]YearWeek, 0, numWeeks)
	for i := 0; i < numWeeks; i++ {
		out = append(out, YearWeek{Year: y, Week: w})
		y, w = nextISOWeek(y, w)
	}
	return out
}

func nextISOWeek(year, week int) (int, int) {
	return mondayOfISOWeek(year, week).AddDate(0, 0, 7).ISOWeek()
}

// mondayOfISOWeek returns the Monday 12:00 UTC of the given ISO week.
func mondayOfISOWeek(year, week int) time.Time {
	// ISO week 1 contains January 4.
	jan4 := time.Date(year, time.January, 4, 12, 0, 0, 0, time.UTC)
	wd := int(jan4.Weekday())
	if wd == 0 {
		wd = 7 // Sunday
	}
	mondayWeek1 := jan4.AddDate(0, 0, 1-wd)
	return mondayWeek1.AddDate(0, 0, (week-1)*7)
}

// FetchWeeks loads menus for slug across the ISO weeks covering from…+numWeeks.
// childNames are attached to each returned SchoolWeek for digest attribution.
func (c *Client) FetchWeeks(ctx context.Context, slug string, childNames []string, from time.Time, numWeeks int, loc *time.Location) ([]SchoolWeek, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("skolmaten slug is empty")
	}
	if c == nil {
		return nil, fmt.Errorf("skolmaten client is nil")
	}
	if strings.TrimSpace(c.token) == "" {
		return nil, fmt.Errorf("skolmaten client token is empty")
	}
	weeks := ISOWeeksCovering(from, numWeeks, loc)
	out := make([]SchoolWeek, 0, len(weeks))
	for _, yw := range weeks {
		sw, err := c.fetchWeek(ctx, slug, yw.Year, yw.Week)
		if err != nil {
			return nil, err
		}
		sw.ChildNames = append([]string(nil), childNames...)
		out = append(out, sw)
	}
	return out, nil
}

func (c *Client) fetchWeek(ctx context.Context, slug string, year, week int) (SchoolWeek, error) {
	sw, status, _, err := c.doMenuRequest(ctx, slug, year, week)
	if err == nil {
		return sw, nil
	}
	if !isAuthStatus(status) || c.refreshed {
		return SchoolWeek{}, err
	}
	newToken, discErr := DiscoverWebClientToken(ctx, c.hc, c.siteBase)
	if discErr != nil {
		return SchoolWeek{}, fmt.Errorf("%w (token refresh failed: %v)", err, discErr)
	}
	if newToken == "" || newToken == c.token {
		return SchoolWeek{}, fmt.Errorf("%w (token refresh found no usable new token)", err)
	}
	c.token = newToken
	c.refreshed = true
	sw, _, _, retryErr := c.doMenuRequest(ctx, slug, year, week)
	if retryErr != nil {
		return SchoolWeek{}, fmt.Errorf("%w (retry after token refresh failed: %v)", err, retryErr)
	}
	return sw, nil
}

func (c *Client) doMenuRequest(ctx context.Context, slug string, year, week int) (SchoolWeek, int, []byte, error) {
	base := c.apiBase
	if base == "" {
		base = defaultAPIBase
	}
	u := fmt.Sprintf("%s/menu/school/%s?year=%d&week=%d",
		base, url.PathEscape(slug), year, week)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return SchoolWeek{}, 0, nil, fmt.Errorf("skolmaten request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Client-Token", c.token)
	req.Header.Set("Referer", "https://skolmaten.se/")
	req.Header.Set("User-Agent", "gmail-planner/1.0")

	resp, err := c.hc.Do(req)
	if err != nil {
		return SchoolWeek{}, 0, nil, fmt.Errorf("skolmaten http %s: %w", slug, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return SchoolWeek{}, resp.StatusCode, nil, fmt.Errorf("skolmaten read %s: %w", slug, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SchoolWeek{}, resp.StatusCode, body, fmt.Errorf("skolmaten status %s for %s week %d-%d: %s",
			resp.Status, slug, year, week, truncate(string(body), 200))
	}
	sw, err := ParseMenuResponse(body, slug)
	if err != nil {
		return SchoolWeek{}, resp.StatusCode, body, fmt.Errorf("skolmaten parse %s: %w", slug, err)
	}
	if sw.Year == 0 {
		sw.Year = year
	}
	if sw.Week == 0 {
		sw.Week = week
	}
	return sw, resp.StatusCode, body, nil
}

func isAuthStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

// DiscoverWebClientToken loads the skolmaten.se SPA bundle and extracts VITE_CLIENT_TOKEN.
// siteBase defaults to https://skolmaten.se when empty.
func DiscoverWebClientToken(ctx context.Context, hc *http.Client, siteBase string) (string, error) {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	siteBase = strings.TrimRight(strings.TrimSpace(siteBase), "/")
	if siteBase == "" {
		siteBase = defaultSiteBase
	}

	homeBody, err := getBody(ctx, hc, siteBase+"/")
	if err != nil {
		return "", fmt.Errorf("fetch skolmaten home: %w", err)
	}
	scriptPath, err := findIndexScriptPath(string(homeBody))
	if err != nil {
		return "", err
	}
	scriptURL := resolveURL(siteBase, scriptPath)
	jsBody, err := getBody(ctx, hc, scriptURL)
	if err != nil {
		return "", fmt.Errorf("fetch skolmaten bundle: %w", err)
	}
	token, err := extractClientToken(string(jsBody))
	if err != nil {
		return "", err
	}
	return token, nil
}

func findIndexScriptPath(html string) (string, error) {
	m := indexScriptRE.FindStringSubmatch(html)
	if len(m) < 2 {
		return "", fmt.Errorf("skolmaten home: no /assets/index-*.js script found")
	}
	return m[1], nil
}

func extractClientToken(js string) (string, error) {
	if m := viteTokenRE.FindStringSubmatch(js); len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
		return strings.TrimSpace(m[1]), nil
	}
	if m := headerTokenRE.FindStringSubmatch(js); len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
		return strings.TrimSpace(m[1]), nil
	}
	return "", fmt.Errorf("skolmaten bundle: client token not found")
}

func resolveURL(siteBase, path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return strings.TrimRight(siteBase, "/") + path
	}
	return strings.TrimRight(siteBase, "/") + "/" + path
}

func getBody(ctx context.Context, hc *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gmail-planner/1.0")
	req.Header.Set("Accept", "*/*")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %s: %s", resp.Status, truncate(string(body), 120))
	}
	return body, nil
}

// ParseMenuResponse extracts school name, week, and day meal names from a skolmaten API JSON body.
func ParseMenuResponse(body []byte, slug string) (SchoolWeek, error) {
	var raw menuAPIResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return SchoolWeek{}, err
	}
	sw := SchoolWeek{
		Slug:       slug,
		SchoolName: strings.TrimSpace(raw.School.Name),
		Year:       raw.WeekState.Year,
		Week:       raw.WeekState.Week,
	}
	if sw.SchoolName == "" {
		sw.SchoolName = strings.TrimSpace(raw.Name)
	}
	for _, d := range raw.WeekState.Days {
		date := normalizeAPIDate(d.Date)
		meals := make([]string, 0, len(d.Meals))
		for _, m := range d.Meals {
			name := strings.TrimSpace(m.Name)
			if name != "" {
				meals = append(meals, name)
			}
		}
		if date == "" && len(meals) == 0 {
			continue
		}
		sw.Days = append(sw.Days, DayMenu{Date: date, Meals: meals})
	}
	return sw, nil
}

type menuAPIResponse struct {
	Name      string `json:"name"`
	WeekState struct {
		Week int `json:"week"`
		Year int `json:"year"`
		Days []struct {
			Date  string `json:"date"`
			Meals []struct {
				Name string `json:"name"`
			} `json:"Meals"`
		} `json:"Days"`
	} `json:"WeekState"`
	School struct {
		Name string `json:"name"`
	} `json:"School"`
}

func normalizeAPIDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// API returns e.g. "2026-11-24T00:00:00.000Z"
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01-02")
	}
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
