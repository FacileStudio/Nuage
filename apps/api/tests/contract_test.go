package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const contractGolden = "testdata/contract.golden"

type volatileValue struct {
	pattern *regexp.Regexp
	stand   string
}

func volatileValues() []volatileValue {
	return []volatileValue{
		{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`), "<time>"},
		{regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "<uuid>"},
		{regexp.MustCompile(`fac_[A-Za-z0-9_-]+`), "fac_<id>"},
		{regexp.MustCompile(`"token":"[^"]+"`), `"token":"<token>"`},
		{regexp.MustCompile(`/shared/[A-Za-z0-9_-]{16,}`), "/shared/<token>"},
		{regexp.MustCompile(`"hash":"[0-9a-f]{64}"`), `"hash":"<sha256>"`},
		{regexp.MustCompile(`"color":"[0-9A-Fa-f]{6}"`), `"color":"<color>"`},
	}
}

func contractHeaders() []string {
	return []string{"Content-Type", "Content-Length", "Content-Disposition", "Cache-Control", "ETag", "Accept-Ranges"}
}

type contractCall struct {
	method, path, token string
	body                any
	raw                 []byte
}

type contractRecorder struct {
	t     *testing.T
	ts    *testServer
	lines []string
}

func (c *contractRecorder) do(call contractCall) []byte {
	c.t.Helper()
	var reader io.Reader
	switch {
	case call.raw != nil:
		reader = bytes.NewReader(call.raw)
	case call.body != nil:
		encoded, err := json.Marshal(call.body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(call.method, call.path, reader)
	if call.body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if call.token != "" {
		request.Header.Set("Authorization", "Bearer "+call.token)
	}
	recorder := httptest.NewRecorder()
	c.ts.router.ServeHTTP(recorder, request)
	body := recorder.Body.Bytes()
	c.lines = append(c.lines, fmt.Sprintf("%s %s -> %d", call.method, normalize(call.path), recorder.Code))
	for _, name := range contractHeaders() {
		if value := recorder.Header().Get(name); value != "" {
			c.lines = append(c.lines, fmt.Sprintf("  %s: %s", name, normalize(value)))
		}
	}
	c.lines = append(c.lines, "  "+normalize(strings.TrimRight(string(body), "\n")))
	return body
}

func normalize(text string) string {
	for _, rule := range volatileValues() {
		text = rule.pattern.ReplaceAllString(text, rule.stand)
	}
	return text
}

func field(t *testing.T, body []byte, path ...string) string {
	t.Helper()
	var decoded any
	require.NoError(t, json.Unmarshal(body, &decoded), string(body))
	for _, name := range path {
		object, ok := decoded.(map[string]any)
		require.True(t, ok, string(body))
		decoded = object[name]
	}
	return fmt.Sprint(decoded)
}

func playContract(c *contractRecorder, owner, other string) {
	t := c.t
	init := c.do(contractCall{method: "POST", path: "/files/upload/init", token: owner,
		body: map[string]any{"file_name": "notes.txt", "total_size": 11}})
	session := field(t, init, "session_id")
	c.do(contractCall{method: "PUT", path: "/files/upload/" + session + "/part/1", token: owner, raw: []byte("hello world")})
	c.do(contractCall{method: "GET", path: "/files/upload/" + session + "/status", token: owner})
	file := field(t, c.do(contractCall{method: "POST", path: "/files/upload/" + session + "/complete", token: owner}), "file", "id")
	c.do(contractCall{method: "GET", path: "/files/" + file, token: owner})
	c.do(contractCall{method: "GET", path: "/files/" + file + "/download", token: owner})
	c.do(contractCall{method: "GET", path: "/files/" + file + "/download", token: other})
	c.do(contractCall{method: "GET", path: "/files", token: owner})
	share := field(t, c.do(contractCall{method: "POST", path: "/shares", token: owner,
		body: map[string]any{"file_id": mustInt(t, file), "permission": "view"}}), "token")
	c.do(contractCall{method: "GET", path: "/shared/" + share})
	c.do(contractCall{method: "GET", path: "/shared/" + share + "/download/" + file})
	c.do(contractCall{method: "GET", path: "/shares/by-me", token: owner})
	c.do(contractCall{method: "GET", path: "/quota/me", token: owner})
	c.do(contractCall{method: "DELETE", path: "/files/" + file, token: owner})
	c.do(contractCall{method: "GET", path: "/trash", token: owner})
	c.do(contractCall{method: "POST", path: "/trash/file/" + file + "/restore", token: owner})
	c.do(contractCall{method: "GET", path: "/sync/state", token: owner})
	c.do(contractCall{method: "DELETE", path: "/files/" + file, token: owner})
	c.do(contractCall{method: "DELETE", path: "/trash/file/" + file, token: owner})
	c.do(contractCall{method: "GET", path: "/quota/me", token: owner})
	c.do(contractCall{method: "GET", path: "/users/me", token: owner})
	c.do(contractCall{method: "GET", path: "/files"})
	c.do(contractCall{method: "GET", path: "/trash"})
	c.do(contractCall{method: "GET", path: "/quota/me"})
}

func mustInt(t *testing.T, text string) int64 {
	t.Helper()
	var n int64
	_, err := fmt.Sscan(text, &n)
	require.NoError(t, err)
	return n
}

func TestTheClientContractStaysByteIdentical(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner@example.com", "")
	_, other := registerUser(ts, "other@example.com", "")
	recorder := &contractRecorder{t: t, ts: ts}
	playContract(recorder, owner, other)
	got := strings.Join(recorder.lines, "\n") + "\n"

	if os.Getenv("NUAGE_WRITE_CONTRACT") != "" {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(contractGolden, []byte(got), 0o644))
	}
	want, err := os.ReadFile(contractGolden)
	require.NoError(t, err)
	assert.Equal(t, string(want), got)
}
