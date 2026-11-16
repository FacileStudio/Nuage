package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestSecurityConfigAllowsNoExternalScript(t *testing.T) {
	cfg := securityConfig(t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, source := range cfg.CSPSources["script-src"] {
		if strings.HasPrefix(source, "https://") {
			t.Errorf("script-src allows the remote origin %s", source)
		}
	}
	if len(cfg.CSPSources["connect-src"]) == 0 {
		t.Error("connect-src lost the Ardoise plan catalog")
	}
}
