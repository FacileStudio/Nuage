package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/FacileStudio/porte/directory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBulle struct {
	server *httptest.Server
	mu     sync.Mutex
	paths  []string
}

func newFakeBulle(t *testing.T) *fakeBulle {
	t.Helper()
	fake := &fakeBulle{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fake.mu.Lock()
		fake.paths = append(fake.paths, strings.TrimSpace(r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery+" "+string(body)))
		fake.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"ideas":[{"id":"i1","kind":"task","title":"Tidy","why":"Because"}],"generating":false}`))
		default:
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"batch_id":"b1","run_id":"r1"}`))
		}
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeBulle) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

func grantAI(t *testing.T, e planEnv, revision int64) {
	t.Helper()
	ai := map[string]json.RawMessage{"ai.access": json.RawMessage("true")}
	snap := directory.Snapshot{
		Version: revision, EntitlementsRevision: revision,
		People: []directory.Person{{Subject: "sub-1", Entitlements: ai}},
		Spaces: []directory.Space{{ID: "spc_1", Name: "Team", Entitlements: ai}},
	}
	require.NoError(t, e.ts.plans.Apply(context.Background(), snap))
}

func TestIdeasRequireASession(t *testing.T) {
	ts := setupTestServer(t)
	for _, path := range []string{"/ideas", "/ideas/runs/r1"} {
		assert.Equal(t, http.StatusUnauthorized, doGet(ts, path, "").StatusCode, path)
	}
	assert.Equal(t, http.StatusUnauthorized, doJSON(ts, "POST", "/ideas/generate", nil, "").StatusCode)
	assert.Empty(t, ts.bulle.calls())
}

func TestIdeasFreePlanSeesNoAIAndNeverReachesBulle(t *testing.T) {
	e := newPlanEnv(t, "free@example.com")
	e.enforce(t, 1, "5", "")

	resp := doGet(e.ts, "/ideas", e.token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var page struct {
		AIAccess   bool   `json:"ai_access"`
		UpgradeURL string `json:"upgrade_url"`
		Ideas      []any  `json:"ideas"`
	}
	parseJSON(resp, &page)
	assert.False(t, page.AIAccess)
	assert.Equal(t, testIssuer+"/spaces", page.UpgradeURL)
	assert.Empty(t, page.Ideas)

	assert.Equal(t, http.StatusPaymentRequired, doJSON(e.ts, "POST", "/ideas/generate", nil, e.token).StatusCode)
	assert.Equal(t, http.StatusPaymentRequired, doJSON(e.ts, "POST", "/ideas/i1/run", nil, e.token).StatusCode)
	assert.Empty(t, e.ts.bulle.calls())
}

func TestIdeasPaidPlanPersonalSendsNoSpace(t *testing.T) {
	e := newPlanEnv(t, "paid@example.com")
	grantAI(t, e, 1)

	resp := doGet(e.ts, "/ideas", e.token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var page struct {
		AIAccess bool  `json:"ai_access"`
		Ideas    []any `json:"ideas"`
	}
	parseJSON(resp, &page)
	assert.True(t, page.AIAccess)
	assert.Len(t, page.Ideas, 1)
	assert.Equal(t, []string{"GET /api/ideas?app=nuage"}, e.ts.bulle.calls())
}

func TestIdeasPassTheRegistreSpaceThrough(t *testing.T) {
	e := newPlanEnv(t, "space@example.com")
	grantAI(t, e, 1)
	spaceID := e.linkedSpace(t)

	resp := doJSON(e.ts, "POST", "/ideas/generate?space_id="+itoa(spaceID), nil, e.token)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Equal(t, []string{`POST /api/ideas/generate?app=nuage {"space_id":"spc_1"}`}, e.ts.bulle.calls())
}

func TestIdeasRefuseSpacesTheCallerCannotUse(t *testing.T) {
	e := newPlanEnv(t, "owner@example.com")
	grantAI(t, e, 1)
	spaceID := e.linkedSpace(t)
	_, stranger := registerUser(e.ts, "stranger@example.com", "password12345")
	unlinked := createSpace(t, e.ts, e.token, "Unlinked")

	assert.Equal(t, http.StatusForbidden, doGet(e.ts, "/ideas?space_id="+itoa(spaceID), stranger).StatusCode)
	assert.Equal(t, http.StatusBadRequest, doGet(e.ts, "/ideas?space_id=abc", e.token).StatusCode)
	assert.Equal(t, http.StatusPreconditionFailed, doGet(e.ts, "/ideas?space_id="+itoa(unlinked), e.token).StatusCode)
	assert.Empty(t, e.ts.bulle.calls())
}

func TestIdeasSpacePlanDecidesNotThePerson(t *testing.T) {
	e := newPlanEnv(t, "mixed@example.com")
	ai := map[string]json.RawMessage{"ai.access": json.RawMessage("true")}
	snap := directory.Snapshot{
		Version: 1, EntitlementsRevision: 1,
		People: []directory.Person{{Subject: "sub-1", Entitlements: ai}},
		Spaces: []directory.Space{{ID: "spc_1", Name: "Team"}},
	}
	require.NoError(t, e.ts.plans.Apply(context.Background(), snap))
	spaceID := e.linkedSpace(t)

	assert.Equal(t, http.StatusPaymentRequired, doJSON(e.ts, "POST", "/ideas/generate?space_id="+itoa(spaceID), nil, e.token).StatusCode)
	assert.Empty(t, e.ts.bulle.calls())
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
