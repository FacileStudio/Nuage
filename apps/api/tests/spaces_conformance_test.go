package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/spaces"
	"github.com/FacileStudio/porte/spaces/spacestest"

	"gorm.io/gorm"
)

func TestSpaceStoreConformance(t *testing.T) {
	ts := setupTestServer(t)
	spacestest.Conformance(t, func() spaces.Store {
		return &fixtureStore{
			store:      spaceaccess.NewStore(ts.db),
			db:         ts.db,
			t:          t,
			spaceIDs:   map[string]int64{},
			userIDs:    map[string]int64{},
			spaceNames: map[string]string{},
			userNames:  map[string]string{},
		}
	})
}

// fixtureStore runs the conformance suite against the real spaceaccess.Store.
// The suite names its fixture "space-a" and "user-owner", while Nuage's
// columns are bigint, so this wrapper allocates a real spaces/users row per
// name and translates in both directions. The reverse translation reads what
// the store returned, never what was asked for, so a store that blanks an id
// or lists the wrong row still fails the suite.
type fixtureStore struct {
	store      *spaceaccess.Store
	db         *gorm.DB
	t          *testing.T
	spaceIDs   map[string]int64
	userIDs    map[string]int64
	spaceNames map[string]string
	userNames  map[string]string
}

func (f *fixtureStore) Seed(ctx context.Context, member spaces.Membership) error {
	space := f.ensureSpace(ctx, member.SpaceID)
	user := f.ensureUser(ctx, member.UserID)
	row := &schemas.SpaceMember{SpaceID: space, UserID: user, Role: string(member.Role)}
	return f.db.WithContext(ctx).Create(row).Error
}

func (f *fixtureStore) Membership(ctx context.Context, spaceID, userID string) (spaces.Membership, error) {
	got, err := f.store.Membership(ctx, f.spaceRef(spaceID), f.userRef(userID))
	if err != nil {
		return spaces.Membership{}, err
	}
	return f.rename(got), nil
}

func (f *fixtureStore) Memberships(ctx context.Context, userID string) ([]spaces.Membership, error) {
	rows, err := f.store.Memberships(ctx, f.userRef(userID))
	if err != nil {
		return nil, err
	}
	out := make([]spaces.Membership, 0, len(rows))
	for _, row := range rows {
		out = append(out, f.rename(row))
	}
	return out, nil
}

func (f *fixtureStore) CountRole(ctx context.Context, spaceID string, role spaces.Role) (int, error) {
	return f.store.CountRole(ctx, f.spaceRef(spaceID), role)
}

func (f *fixtureStore) ensureSpace(ctx context.Context, name string) int64 {
	if id, ok := f.spaceIDs[name]; ok {
		return id
	}
	space := &schemas.Space{FacileID: facile.NewID(), Name: name}
	if err := f.db.WithContext(ctx).Create(space).Error; err != nil {
		f.t.Fatalf("seed space %s: %v", name, err)
	}
	f.spaceIDs[name] = space.ID
	f.spaceNames[ref(space.ID)] = name
	return space.ID
}

func (f *fixtureStore) ensureUser(ctx context.Context, name string) int64 {
	if id, ok := f.userIDs[name]; ok {
		return id
	}
	user := &schemas.User{Email: fmt.Sprintf("%s-%s@conformance.test", name, facile.NewID()), Name: name}
	if err := f.db.WithContext(ctx).Create(user).Error; err != nil {
		f.t.Fatalf("seed user %s: %v", name, err)
	}
	f.userIDs[name] = user.ID
	f.userNames[ref(user.ID)] = name
	return user.ID
}

func (f *fixtureStore) spaceRef(name string) string {
	return ref(f.spaceIDs[name])
}

func (f *fixtureStore) userRef(name string) string {
	return ref(f.userIDs[name])
}

func ref(id int64) string {
	return strconv.FormatInt(id, 10)
}

func (f *fixtureStore) rename(member spaces.Membership) spaces.Membership {
	return spaces.Membership{
		SpaceID: f.spaceNames[member.SpaceID],
		UserID:  f.userNames[member.UserID],
		Role:    member.Role,
	}
}
