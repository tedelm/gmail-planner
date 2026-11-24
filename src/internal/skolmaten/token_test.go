package skolmaten

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiscoverWebClientToken_FromViteBundle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<!doctype html><script src="/assets/index-abc123.js"></script>`))
	})
	mux.HandleFunc("/assets/index-abc123.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`var x={};VITE_CLIENT_TOKEN:"web-fresh-token-99",y:1`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tok, err := DiscoverWebClientToken(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("DiscoverWebClientToken: %v", err)
	}
	if tok != "web-fresh-token-99" {
		t.Fatalf("token: got %q", tok)
	}
}

func TestDiscoverWebClientToken_FallbackClientTokenHeader(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<script src="/assets/index-x.js"></script>`))
	})
	mux.HandleFunc("/assets/index-x.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`o={"Content-Type":"application/json","Client-Token":"web-header-token"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tok, err := DiscoverWebClientToken(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("DiscoverWebClientToken: %v", err)
	}
	if tok != "web-header-token" {
		t.Fatalf("token: got %q", tok)
	}
}

func TestDiscoverWebClientToken_MissingScript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>no scripts</body></html>`))
	}))
	defer srv.Close()
	_, err := DiscoverWebClientToken(context.Background(), srv.Client(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "index-") {
		t.Fatalf("expected index script error, got %v", err)
	}
}

func TestFetchWeek_ValidTokenNoDiscovery(t *testing.T) {
	var discovered bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Client-Token") != "good-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"name":"Kök","WeekState":{"week":48,"year":2026,"Days":[{"date":"2026-11-24T00:00:00.000Z","Meals":[{"name":"Pasta"}]}]},"School":{"name":"Testskolan"}}`))
	}))
	defer api.Close()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discovered = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer site.Close()

	c := NewClient(api.Client(), "good-token")
	c.apiBase = api.URL
	c.siteBase = site.URL

	menus, err := c.FetchWeeks(context.Background(), "testskolan", []string{"Annie"}, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC), 1, time.UTC)
	if err != nil {
		t.Fatalf("FetchWeeks: %v", err)
	}
	if discovered {
		t.Fatal("expected no SPA discovery with valid token")
	}
	if c.TokenRefreshed() {
		t.Fatal("TokenRefreshed should be false")
	}
	if len(menus) != 1 || len(menus[0].Days) != 1 || menus[0].Days[0].Meals[0] != "Pasta" {
		t.Fatalf("menus: %#v", menus)
	}
}

func TestFetchWeek_AuthThenRefreshRetry(t *testing.T) {
	menuCalls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		menuCalls++
		tok := r.Header.Get("Client-Token")
		if tok != "fresh-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`unauthorized`))
			return
		}
		w.Write([]byte(`{"name":"Kök","WeekState":{"week":48,"year":2026,"Days":[{"date":"2026-11-24T00:00:00.000Z","Meals":[{"name":"Fisk"}]}]},"School":{"name":"Testskolan"}}`))
	}))
	defer api.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<script src="/assets/index-z.js"></script>`))
	})
	mux.HandleFunc("/assets/index-z.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`VITE_CLIENT_TOKEN:"fresh-token"`))
	})
	site := httptest.NewServer(mux)
	defer site.Close()

	c := NewClient(api.Client(), "stale-token")
	c.apiBase = api.URL
	c.siteBase = site.URL
	// Use site client for discovery against site; api.Client() works for both if same host family —
	// discovery hits site.URL, menu hits api.URL; DefaultTransport is fine via api.Client().

	menus, err := c.FetchWeeks(context.Background(), "testskolan", []string{"Annie"}, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC), 1, time.UTC)
	if err != nil {
		t.Fatalf("FetchWeeks: %v", err)
	}
	if !c.TokenRefreshed() {
		t.Fatal("expected TokenRefreshed")
	}
	if menuCalls != 2 {
		t.Fatalf("menuCalls: got %d want 2", menuCalls)
	}
	if len(menus) != 1 || menus[0].Days[0].Meals[0] != "Fisk" {
		t.Fatalf("menus: %#v", menus)
	}
}

func TestFetchWeek_AuthRefreshFails(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`forbidden`))
	}))
	defer api.Close()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>no bundle</html>`))
	}))
	defer site.Close()

	c := NewClient(api.Client(), "stale-token")
	c.apiBase = api.URL
	c.siteBase = site.URL

	_, err := c.FetchWeeks(context.Background(), "testskolan", nil, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC), 1, time.UTC)
	if err == nil || !strings.Contains(err.Error(), "token refresh failed") {
		t.Fatalf("expected refresh failure, got %v", err)
	}
	if c.TokenRefreshed() {
		t.Fatal("should not mark refreshed on failed discovery")
	}
}

func TestExtractClientToken(t *testing.T) {
	tok, err := extractClientToken(`VITE_CLIENT_TOKEN:"abc"`)
	if err != nil || tok != "abc" {
		t.Fatalf("got %q %v", tok, err)
	}
	_, err = extractClientToken(`no token here`)
	if err == nil {
		t.Fatal("expected error")
	}
}
