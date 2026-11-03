package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/porte/directory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRegistre answers the directory API the way an undeployed then deployed Registre
// would: the first import is a 404, every later call succeeds. It records the calls in
// order, with the body of the import that succeeded.
type fakeRegistre struct {
	mu       sync.Mutex
	calls    []string
	imported []directory.ImportSpace
}

func (f *fakeRegistre) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, secret, ok := r.BasicAuth(); !ok || id != "nuage" || secret != "s3cret" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	if r.URL.Path == "/api/directory/import" {
		if len(f.calls) == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Spaces []directory.ImportSpace `json:"spaces"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.imported = body.Spaces
	}
	_ = json.NewEncoder(w).Encode(teamSnapshot())
}

func TestRegistreImportsBeforeTheFirstSync(t *testing.T) {
	ts := setupTestServer(t)
	seedRegistre(t, ts)
	fake := &fakeRegistre{}
	server := httptest.NewServer(fake)
	defer server.Close()

	registre := newTestRegistre(ts.db)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	syncer := &directory.Syncer{
		Client: directory.Client{Issuer: server.URL, ClientID: "nuage", ClientSecret: "s3cret"},
		Apply: func(ctx context.Context, snapshot directory.Snapshot) error {
			defer cancel()
			return registre.Apply(ctx, snapshot)
		},
	}
	registre.Run(ctx, syncer)

	want := []string{"POST /api/directory/import", "POST /api/directory/import", "GET /api/directory/snapshot"}
	require.Equal(t, want, fake.calls, "retry the import, then sync")
	require.Len(t, fake.imported, 1, "the provisioned space stays out: %+v", fake.imported)
	assert.Equal(t, "team", fake.imported[0].Name)
	wantMembers := []directory.ImportMember{
		{Subject: "sub-alice", Role: directory.RoleOwner},
		{Email: "carol@x.test", Role: directory.RoleAdmin},
		{Subject: "sub-dave", Role: directory.RoleAdmin},
	}
	got := slices.Clone(fake.imported[0].Members)
	slices.SortFunc(got, func(a, b directory.ImportMember) int { return compareMember(a) - compareMember(b) })
	assert.Equal(t, wantMembers, got, "subject when known, else email")
	assert.Equal(t, "admin", roleIn(membershipState(t, ts.db), "r-team:Team/bob@x.test"), "the sync after the import applied")
}

// compareMember orders the seeded members alice, carol, dave, whatever order they joined in.
func compareMember(member directory.ImportMember) int {
	return map[string]int{"sub-alice": 0, "carol@x.test": 1, "sub-dave": 2}[member.Subject+member.Email]
}
