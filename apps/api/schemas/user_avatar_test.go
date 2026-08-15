package schemas

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDatabase hands back a connection scoped to a schema of its own, so these tests
// can create and drop `users` without disturbing the integration suite sharing the
// database. The backfill is written in PostgreSQL (regexp_replace, anchored), so testing
// it anywhere else would test a different statement than the one that ships.
//
// A developer without Postgres still gets a usable suite; CI, where the
// database is always there, would fail on the same condition rather than
// skip silently.
func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://nuage:nuage-internal-db@localhost:5432/nuage_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("integration test infrastructure required in CI: %v", err)
		}
		t.Skipf("skipping: database not available: %v", err)
	}

	schema := fmt.Sprintf("avatar_test_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("integration test infrastructure required in CI: %v", err)
		}
		t.Skipf("skipping: database not available: %v", err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA " + schema + " CASCADE") })

	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return db
}

func TestAvatarPrecedence(t *testing.T) {
	const porte = "https://porte.facile.studio/media/user-avatars/x.png"

	cases := []struct {
		name       string
		user       User
		wantURL    string
		wantOrigin string
	}{
		{"Porte photo wins over an upload", User{OIDCPictureURL: porte, AvatarUploadPath: "user-3-1.png"}, porte, "oidc"},
		{"upload is the fallback", User{AvatarUploadPath: "user-3-1.png"}, "/api/avatars/user-3-1.png", "upload"},
		{"only Porte", User{OIDCPictureURL: porte}, porte, "oidc"},
		{"neither, so the client draws initials", User{}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.user.Avatar(); got != tc.wantURL {
				t.Errorf("Avatar() = %q, want %q", got, tc.wantURL)
			}
			if got := tc.user.AvatarOrigin(); got != tc.wantOrigin {
				t.Errorf("AvatarOrigin() = %q, want %q", got, tc.wantOrigin)
			}
		})
	}
}

// The same rule is spelled twice, once in Go and once in SQL for the joins that read an
// avatar without loading the row. This is the test that fails when someone edits one and
// forgets the other.
func TestAvatarSelectExprMatchesAvatar(t *testing.T) {
	orm := openTestDatabase(t)

	users := []User{
		{Email: "both@example.com", OIDCPictureURL: "https://porte.facile.studio/media/user-avatars/a.png", AvatarUploadPath: "user-1-1.png"},
		{Email: "upload@example.com", AvatarUploadPath: "user-2-1.png"},
		{Email: "oidc@example.com", OIDCPictureURL: "https://porte.facile.studio/media/user-avatars/b.png"},
		{Email: "neither@example.com"},
	}
	for i := range users {
		if err := orm.Create(&users[i]).Error; err != nil {
			t.Fatalf("create %s: %v", users[i].Email, err)
		}
	}

	for _, want := range users {
		var got string
		if err := orm.Model(&User{}).
			Select(AvatarSelectExpr).
			Where("users.id = ?", want.ID).
			Scan(&got).Error; err != nil {
			t.Fatalf("select for %s: %v", want.Email, err)
		}
		if got != want.Avatar() {
			t.Errorf("%s: SQL gave %q, Avatar() gave %q", want.Email, got, want.Avatar())
		}
	}
}

// Rows 2 and 3 are the reason this test exists. Row 2 holds an uploaded avatar with
// avatar_source empty because it predates that column, and a backfill keyed on
// the 'upload' source would drop its picture without a word. Row 4 carries the data:
// URI the old sync stored verbatim for every user Authentik has no photo for; left alone
// it would read as "there is an SSO photo" and suppress the upload fallback forever.
//
// The row that carries both keeps its file and still renders the Porte
// photo, and the placeholder row is gone so its user can upload again
// and see it.
func TestBackfillAvatarUploadPath(t *testing.T) {
	orm := openTestDatabase(t)

	rows := []struct {
		email      string
		url        string
		source     string
		oidc       string
		wantUpload string
		wantOIDC   string
	}{
		{"oidc-copy@example.com", "/api/avatars/oidc-1-1780064725810006439.png", "oidc", "https://porte.facile.studio/media/user-avatars/a.png", "", "https://porte.facile.studio/media/user-avatars/a.png"},
		{"legacy-upload@example.com", "/api/avatars/user-2-1778020000000000000.jpg", "", "", "user-2-1778020000000000000.jpg", ""},
		{"upload-and-sso@example.com", "/api/avatars/user-3-1780960000000000000.jpg", "upload", "https://porte.facile.studio/media/user-avatars/b.jpeg", "user-3-1780960000000000000.jpg", "https://porte.facile.studio/media/user-avatars/b.jpeg"},
		{"placeholder@example.com", "", "", "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=", "", ""},
		{"no-avatar@example.com", "", "", "", "", ""},
	}
	for _, row := range rows {
		if err := orm.Exec(
			`INSERT INTO users (email, password_hash, avatar_url, avatar_source, oidc_picture_url) VALUES (?, 'hash', ?, ?, ?)`,
			row.email, row.url, row.source, row.oidc).Error; err != nil {
			t.Fatalf("insert %s: %v", row.email, err)
		}
	}

	if err := backfillAvatarUploadPath(orm); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	for _, row := range rows {
		var got User
		if err := orm.Where("email = ?", row.email).First(&got).Error; err != nil {
			t.Fatalf("read %s: %v", row.email, err)
		}
		if got.AvatarUploadPath != row.wantUpload {
			t.Errorf("%s: avatar_upload_path = %q, want %q", row.email, got.AvatarUploadPath, row.wantUpload)
		}
		if got.OIDCPictureURL != row.wantOIDC {
			t.Errorf("%s: oidc_picture_url = %q, want %q", row.email, got.OIDCPictureURL, row.wantOIDC)
		}
	}

	var both User
	if err := orm.Where("email = ?", "upload-and-sso@example.com").First(&both).Error; err != nil {
		t.Fatalf("read both: %v", err)
	}
	if both.Avatar() != "https://porte.facile.studio/media/user-avatars/b.jpeg" {
		t.Errorf("SSO photo should win, got %q", both.Avatar())
	}

	var placeholder User
	if err := orm.Where("email = ?", "placeholder@example.com").First(&placeholder).Error; err != nil {
		t.Fatalf("read placeholder: %v", err)
	}
	if placeholder.AvatarOrigin() != "" {
		t.Errorf("a data: placeholder must not count as an SSO photo, origin = %q", placeholder.AvatarOrigin())
	}
}
