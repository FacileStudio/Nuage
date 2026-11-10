package tests

import (
	"bytes"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/modules/mcp"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestMCPConformance runs tronc's conformance kit, with trashing and
// sharing as the sensitive tools.
func TestMCPConformance(t *testing.T) {
	f := newMCPFixture(t)
	fresh := func(runs func(int64) int) func(testing.TB) (map[string]any, func() int) {
		return func(tb testing.TB) (map[string]any, func() int) {
			file := f.upload(tb, f.owner, tb.Name()+".txt", tb.Name(), nil)
			return map[string]any{"type": "file", "id": file.ID}, func() int { return runs(file.ID) }
		}
	}
	mcptest.Run(t, mcptest.Config{
		URL:   f.resource,
		Token: f.token,
		Confirm: []mcptest.ConfirmCase{
			{Tool: "nuage_delete", Setup: fresh(f.trashed)},
			{Tool: "nuage_create_share", Setup: fresh(f.shares)},
		},
	})
}

// TestMCPSaveListSearchRead walks a model's usual path: make a folder,
// save a note in it, find it and read it back.
func TestMCPSaveListSearchRead(t *testing.T) {
	f := newMCPFixture(t)
	session := f.connect(t, "owner", "")
	var folder, again struct {
		Status string             `json:"status"`
		Item   struct{ ID int64 } `json:"item"`
	}
	call(t, session, "nuage_create_folder", map[string]any{"name": "Clients"}, &folder)
	call(t, session, "nuage_create_folder", map[string]any{"name": "Clients"}, &again)
	if folder.Status != "created" || again.Status != "exists" || again.Item.ID != folder.Item.ID {
		t.Fatalf("a second create must return the same folder, got %+v then %+v", folder, again)
	}
	var saved struct {
		Status string `json:"status"`
		Item   struct {
			ID       int64  `json:"id"`
			MimeType string `json:"mime_type"`
		} `json:"item"`
	}
	call(t, session, "nuage_save_file", map[string]any{"name": "acme.md", "content": "# Acme\nélan", "folder_id": folder.Item.ID}, &saved)
	if saved.Status != "created" || !strings.HasPrefix(saved.Item.MimeType, "text/markdown") {
		t.Fatalf("want a markdown file, got %+v", saved)
	}
	var root struct {
		Items  []struct{ Name, Type string } `json:"items"`
		Spaces *[]any                        `json:"spaces"`
	}
	call(t, session, "nuage_list_folder", nil, &root)
	if len(root.Items) != 1 || root.Items[0].Name != "Clients" || root.Spaces == nil {
		t.Fatalf("the root must hold the folder and list spaces, got %+v", root)
	}
	var found struct {
		Items []struct {
			ID   int64
			Path string
		} `json:"items"`
	}
	call(t, session, "nuage_search", map[string]any{"query": "ACME"}, &found)
	if len(found.Items) != 1 || found.Items[0].Path != "/Clients/acme.md" {
		t.Fatalf("search must find the note with its path, got %+v", found)
	}
	var read struct {
		Content string `json:"content"`
		Text    bool   `json:"text"`
	}
	call(t, session, "nuage_read_file", map[string]any{"file_id": saved.Item.ID}, &read)
	if !read.Text || read.Content != "# Acme\nélan" {
		t.Fatalf("read must return the note, got %+v", read)
	}
}

// TestMCPReadPagesALongFile reads a file bigger than one call by offsets,
// with a multibyte rune on every chunk boundary candidate.
func TestMCPReadPagesALongFile(t *testing.T) {
	f := newMCPFixture(t)
	content := strings.Repeat("é<b>&\"line\"\n", 15_000)
	file := f.upload(t, f.owner, "long.txt", content, nil)
	session := f.connect(t, "owner", "")
	var whole strings.Builder
	offset := int64(0)
	for calls := 0; calls < 20; calls++ {
		var out struct {
			Content    string `json:"content"`
			Offset     int64  `json:"offset"`
			NextOffset int64  `json:"next_offset"`
		}
		result := mcptest.Call(t, session, "nuage_read_file", map[string]any{"file_id": file.ID, "offset": offset})
		if size := len(mcptest.Text(result)); size > mcpkit.ResultBudget*2 {
			t.Fatalf("a read result must stay near the budget, got %d bytes", size)
		}
		mcptest.Decode(t, result, &out)
		whole.WriteString(out.Content)
		if out.NextOffset == 0 {
			break
		}
		offset = out.NextOffset
	}
	if whole.String() != content {
		t.Fatalf("the pages must add up to the file: got %d bytes of %d", whole.Len(), len(content))
	}
}

// TestMCPBinaryIsNotRead checks a binary file comes back without content.
func TestMCPBinaryIsNotRead(t *testing.T) {
	f := newMCPFixture(t)
	file := f.upload(t, f.owner, "blob.bin", "PK\x03\x04\x00\x00binary", nil)
	var out struct {
		Text    bool   `json:"text"`
		Content string `json:"content"`
	}
	call(t, f.connect(t, "owner", ""), "nuage_read_file", map[string]any{"file_id": file.ID}, &out)
	if out.Text || out.Content != "" {
		t.Fatalf("a binary file must not be read, got %+v", out)
	}
}

// TestMCPRespectsSpaces checks a stranger reaches nothing of a space until
// they belong to it, exactly like the REST API.
func TestMCPRespectsSpaces(t *testing.T) {
	f := newMCPFixture(t)
	space := schemas.Space{FacileID: "space-mcp", Name: "Studio"}
	if err := f.ts.db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.ts.db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: f.owner, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	file := f.upload(t, f.owner, "plan.txt", "secret plan", &space.ID)
	stranger := f.connect(t, "stranger", "")
	refused(t, stranger, "nuage_read_file", map[string]any{"file_id": file.ID})
	refused(t, stranger, "nuage_list_folder", map[string]any{"space_id": space.ID})
	refused(t, stranger, "nuage_delete", map[string]any{"type": "file", "id": file.ID})
	refused(t, stranger, "nuage_create_share", map[string]any{"type": "file", "id": file.ID})
	var found struct {
		Items []any `json:"items"`
	}
	call(t, stranger, "nuage_search", map[string]any{"query": "plan"}, &found)
	if len(found.Items) != 0 {
		t.Fatalf("a stranger must not find space files, got %+v", found)
	}
	if err := f.ts.db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: f.stranger, Role: "member"}).Error; err != nil {
		t.Fatal(err)
	}
	var read struct {
		Content string `json:"content"`
	}
	call(t, stranger, "nuage_read_file", map[string]any{"file_id": file.ID}, &read)
	call(t, stranger, "nuage_search", map[string]any{"query": "plan"}, &found)
	if read.Content != "secret plan" || len(found.Items) != 1 {
		t.Fatalf("a member reads and finds space files, got %q and %+v", read.Content, found)
	}
	if text := refused(t, stranger, "nuage_save_file", map[string]any{"file_id": file.ID, "content": "mine now"}); !strings.Contains(text, "uploaded") {
		t.Fatalf("only the uploader overwrites, got %q", text)
	}
}

// TestMCPOverwriteKeepsAVersion checks file_id overwrites and keeps the
// old content as a version.
func TestMCPOverwriteKeepsAVersion(t *testing.T) {
	f := newMCPFixture(t)
	file := f.upload(t, f.owner, "todo.txt", "v1", nil)
	session := f.connect(t, "owner", "")
	var out struct {
		Status string `json:"status"`
	}
	call(t, session, "nuage_save_file", map[string]any{"file_id": file.ID, "content": "v2"}, &out)
	var read struct {
		Content string `json:"content"`
	}
	call(t, session, "nuage_read_file", map[string]any{"file_id": file.ID}, &read)
	var versions int64
	f.ts.db.Model(&schemas.FileVersion{}).Where("file_id = ?", file.ID).Count(&versions)
	if out.Status != "replaced" || read.Content != "v2" || versions != 1 {
		t.Fatalf("want v2 with one version kept, got %+v, %q, %d versions", out, read.Content, versions)
	}
}

// TestMCPActorsFollowTheAPIRule checks delegated tokens: Bulle acts with
// full access, a registered app and an unknown client reach nothing.
func TestMCPActorsFollowTheAPIRule(t *testing.T) {
	f := newMCPFixture(t)
	var out struct {
		Items []any `json:"items"`
	}
	call(t, f.connect(t, "owner", "bulle"), "nuage_list_folder", nil, &out)
	if text := refused(t, f.connect(t, "owner", "plume"), "nuage_list_folder", nil); !strings.Contains(text, "app folder") {
		t.Fatalf("a registered app must be told it only reaches its app folder, got %q", text)
	}
	refused(t, f.connect(t, "owner", "stranger-client"), "nuage_list_folder", nil)
}

// TestMCPConfirmPageMakesTheLink checks the no-elicitation path end to
// end: the share waits, the person confirms in Nuage, and the link works.
func TestMCPConfirmPageMakesTheLink(t *testing.T) {
	f := newMCPFixture(t)
	file := f.upload(t, f.owner, "deck.txt", "slides", nil)
	var pending struct {
		Status     string `json:"status"`
		ConfirmURL string `json:"confirm_url"`
	}
	call(t, f.connect(t, "owner", ""), "nuage_create_share", map[string]any{"type": "file", "id": file.ID, "expires_in_days": 7}, &pending)
	if pending.Status != mcpkit.StatusPending || f.shares(file.ID) != 0 {
		t.Fatalf("want a pending share and no link yet, got %+v", pending)
	}
	id := pending.ConfirmURL[strings.LastIndex(pending.ConfirmURL, "=")+1:]
	base := strings.TrimSuffix(f.resource, "/mcp")
	req, _ := http.NewRequest(http.MethodPost, base+"/api/mcp/pending/"+id+"/confirm", bytes.NewReader(nil))
	req.Header.Set("Authorization", "Bearer "+f.session)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var result mcp.PendingResult
	parseJSON(resp, &result)
	if resp.StatusCode != http.StatusOK || result.Status != "shared" || !strings.Contains(result.URL, "/s/") || f.shares(file.ID) != 1 {
		t.Fatalf("confirming must make one link, got %d %+v and %d links", resp.StatusCode, result, f.shares(file.ID))
	}
}

// TestMCPServe serves the endpoint over the scratch stack on MCP_SERVE
// until interrupted, for the official conformance suite and MCP
// Inspector. A request without a token gets the owner's, because the
// conformance suite sends none.
func TestMCPServe(t *testing.T) {
	addr := os.Getenv("MCP_SERVE")
	if addr == "" {
		t.Skip("set MCP_SERVE=127.0.0.1:<port> to serve the endpoint")
	}
	f := newMCPFixtureOn(t, addr)
	token := f.token(t)
	f.front = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			next.ServeHTTP(w, r)
		})
	}
	f.upload(t, f.owner, "readme.md", "# Hello", nil)
	t.Logf("serving %s", f.resource)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
