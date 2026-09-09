package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/claudioed/process-path-management/internal/adapters/inbound/auth"
)

func TestBuildAuth_ModeSelection(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		wantMode auth.Mode
		wantWarn bool
	}{
		{"no keys => off + WARN", map[string]string{}, auth.ModeOff, true},
		{"keys => enforce", map[string]string{"API_READ_KEY": "secret-read-0123"}, auth.ModeEnforce, false},
		{"MCP fallback keys => enforce", map[string]string{"MCP_READWRITE_KEY": "secret-mcp-rw-0123"}, auth.ModeEnforce, false},
		{"AUTH_MODE=log overrides", map[string]string{"API_READWRITE_KEY": "secret-rw-0123", "AUTH_MODE": "log"}, auth.ModeLog, false},
		{"AUTH_MODE=off with keys warns", map[string]string{"API_READWRITE_KEY": "secret-rw-0123", "AUTH_MODE": "off"}, auth.ModeOff, true},
		{"AUTH_MODE=enforce without keys still enforces", map[string]string{"AUTH_MODE": "enforce"}, auth.ModeEnforce, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			mw := buildAuth(func(k string) string { return c.env[k] }, logger)
			if mw.Mode != c.wantMode {
				t.Fatalf("want mode %q, got %q", c.wantMode, mw.Mode)
			}
			if got := strings.Contains(buf.String(), "REST auth is OFF"); got != c.wantWarn {
				t.Fatalf("want warn=%v, got %v; log=%s", c.wantWarn, got, buf.String())
			}
			if strings.Contains(buf.String(), "secret-") {
				t.Fatalf("key material must never be logged: %s", buf.String())
			}
		})
	}
}
