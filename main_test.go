package main

import (
	"net/http"
	"net/http/httptest"
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
