package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeHTTPPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback string
		expect   string
	}{
		{name: "empty uses fallback", input: "", fallback: "/health", expect: "/health"},
		{name: "adds slash", input: "manifest", fallback: "/health", expect: "/manifest"},
		{name: "keeps slash", input: "/manifest", fallback: "/health", expect: "/manifest"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeHTTPPath(test.input, test.fallback); got != test.expect {
				t.Fatalf("expected %q, got %q", test.expect, got)
			}
		})
	}
}

func TestSanitizeManifestHostname(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "lowercases", input: "Prod-Logs-01", expect: "prod-logs-01"},
		{name: "replaces unsafe characters", input: "prod logs_01", expect: "prod-logs-01"},
		{name: "fallback", input: "___", expect: "localhost"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sanitizeManifestHostname(test.input); got != test.expect {
				t.Fatalf("expected %q, got %q", test.expect, got)
			}
		})
	}
}

func TestDefaultManifestNameUsesHostname(t *testing.T) {
	cfg := defaultConfig()
	if cfg.ManifestName == legacyManifestName {
		t.Fatalf("expected default manifest name to include hostname, got %q", cfg.ManifestName)
	}
	if !strings.HasPrefix(cfg.ManifestName, "logger.") || !strings.HasSuffix(cfg.ManifestName, "/mcp") {
		t.Fatalf("unexpected default manifest name %q", cfg.ManifestName)
	}
}

func TestLoadConfigRewritesLegacyManifestPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(configPath, []byte(`manifest_path: "/manifest"`), 0600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected load config error: %v", err)
	}
	if cfg.ManifestPath != defaultManifestPath {
		t.Fatalf("expected manifest path %q, got %q", defaultManifestPath, cfg.ManifestPath)
	}
}

func TestExtractAccessKeyFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/sse?access_key=query-key", nil)
	req.Header.Set("Authorization", "Bearer header-key")

	if got := extractAccessKeyFromRequest(req); got != "header-key" {
		t.Fatalf("expected header key to win, got %q", got)
	}
}

func TestIsAccessKeyValid(t *testing.T) {
	cfg := &Config{AccessKey: "secret"}
	if !isAccessKeyValid(cfg, "secret") {
		t.Fatal("expected valid access key")
	}
	if isAccessKeyValid(cfg, "wrong") {
		t.Fatal("expected invalid access key")
	}
}

func TestHealthHandlerReturnsOnlyOK(t *testing.T) {
	cfg := defaultConfig()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(&cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", got)
	}
}

func TestManifestHandlerAddsCORSHeader(t *testing.T) {
	cfg := defaultConfig()
	req := httptest.NewRequest(http.MethodGet, defaultManifestPath, nil)
	rec := httptest.NewRecorder()

	manifestHandler(&cfg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS header %q, got %q", "*", got)
	}
}

func TestLocalCertificateIPsIncludesLoopback(t *testing.T) {
	ips := localCertificateIPs()
	for _, ip := range ips {
		if ip.String() == "127.0.0.1" {
			return
		}
	}
	t.Fatalf("expected certificate IPs to include 127.0.0.1, got %v", ips)
}

func TestWithMostLikelyYear(t *testing.T) {
	now := time.Date(2026, time.January, 1, 1, 0, 0, 0, time.UTC)
	ts := time.Date(0, time.December, 31, 23, 59, 59, 0, time.UTC)

	got := withMostLikelyYear(ts, now)
	if got.Year() != 2025 {
		t.Fatalf("expected previous year for year rollover, got %d", got.Year())
	}
}

func TestParseReadLogsParams(t *testing.T) {
	args := map[string]any{
		"page":       float64(2),
		"page_size":  float64(5000),
		"pattern":    "error*disk",
		"encrypt":    true,
		"start_date": "2026-04-16",
	}

	params, err := parseReadLogsParams(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 2 {
		t.Fatalf("expected page 2, got %d", params.Page)
	}
	if params.PageSize != maxPageSize {
		t.Fatalf("expected page size capped to %d, got %d", maxPageSize, params.PageSize)
	}
	if !params.HasStart {
		t.Fatal("expected start date to be parsed")
	}
	if !params.Encrypt {
		t.Fatal("expected encrypt flag to be true")
	}
	if params.Pattern != "error*disk" {
		t.Fatalf("unexpected pattern %q", params.Pattern)
	}
}
