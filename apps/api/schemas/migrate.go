package schemas

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/Nuage/apps/api/migrations"

	"gorm.io/gorm"
)

// reconcileLockID namespaces the advisory lock that serialises the boot data
// steps across concurrently starting instances. The schema itself is
// serialised by tronc/migrate's own lock.
const reconcileLockID = 4919

// Migrate applies the migrations and then the boot data steps, with no OIDC
// issuer. The tests build their schemas through it, so they run exactly the
// SQL production runs.
func Migrate(db *gorm.DB) error {
	return MigrateWithIssuer(context.Background(), db, "", slog.Default())
}

// MigrateWithIssuer applies the ordered migrations under migrations/ and then
// Reconcile. The issuer is what the identity backfill needs: porte matches an
// account on (provider, subject) and the provider is the issuer, so
// backfilling with a placeholder would leave every existing SSO user
// unmatched and quietly fall through to the email path.
func MigrateWithIssuer(ctx context.Context, db *gorm.DB, issuer string, logger *slog.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := migrations.Up(ctx, migrations.Config(sqlDB, logger)); err != nil {
		return err
	}
	return Reconcile(ctx, db, issuer)
}

// Reconcile runs the idempotent data steps that ran at every boot before the
// schema moved to migrations, in the same order. It changes rows, never the
// schema: the avatar, media type and plan payer backfills, the first
// administrator, and the credentials porte reads. They run in one transaction
// under an advisory lock, so two instances starting together do not both move
// the credentials; the colour backfill runs after it, as it always did.
func Reconcile(ctx context.Context, db *gorm.DB, issuer string) error {
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", reconcileLockID).Error; err != nil {
			return err
		}
		if err := backfill(tx); err != nil {
			return err
		}
		if err := ensureAdmin(tx); err != nil {
			return err
		}
		return AdoptPorte(tx, issuer)
	}); err != nil {
		return err
	}
	return usercolor.BackfillMissing(ctx, db)
}

// backfillAvatarUploadPath moves the uploaded avatars onto the column that now owns them,
// and clears the stored copies of the SSO photo that nothing reads any more.
//
// The filename decides, not avatar_source. That column was added after the upload feature,
// so the oldest uploaded avatars have it empty — on Sablier's production database two of
// the four rows were exactly that, and keying on avatar_source = 'upload' would silently
// have dropped their picture. persistAvatarFile has always named uploads "user-<id>-<nanos>"
// and the old OIDC download named its copies "oidc-<id>-<nanos>", so anything that is not
// an oidc- copy is somebody's upload and is kept.
//
// The prefix strip is anchored. Nuage serves avatars from /api/avatars/, not the /files/
// the rest of the suite uses, and an unanchored replace() would mangle any filename that
// happened to contain the prefix.
//
// backfillAvatarUploadPath back-fills avatar_upload_path and cleans
// oidc_picture_url on rows created by older releases.
//
// avatar_url and avatar_source stay in the table, unread, until a later
// release drops them. Expanding first means a rollback is redeploying the
// old binary rather than restoring a backup.
//
// The old code stored profile.Picture verbatim, so every user without a
// photo in Authentik carries a data: URI of their own initials here. Under
// the new rule this column means "there is an SSO photo", so leaving the
// placeholder would suppress the upload fallback for those users forever.
//
// A NULL in a freshly added column would fail to scan into the plain string
// the model declares.
func backfillAvatarUploadPath(db *gorm.DB) error {
	if db.Migrator().HasColumn(&User{}, "avatar_url") {
		if err := db.Exec(
			`UPDATE users SET avatar_upload_path = regexp_replace(avatar_url, '^/api/avatars/', '')
			 WHERE coalesce(avatar_url, '') <> ''
			   AND avatar_url LIKE '/api/avatars/%'
			   AND avatar_url NOT LIKE '/api/avatars/oidc-%'
			   AND coalesce(avatar_upload_path, '') = ''`).Error; err != nil {
			return err
		}
	}

	if err := db.Exec(
		`UPDATE users SET oidc_picture_url = ''
		 WHERE coalesce(oidc_picture_url, '') <> ''
		   AND lower(oidc_picture_url) NOT LIKE 'https://%'`).Error; err != nil {
		return err
	}

	return db.Exec(`UPDATE users SET avatar_upload_path = '' WHERE avatar_upload_path IS NULL`).Error
}

// backfillMimeTypes gives files stored as application/octet-stream the media
// type their extension implies, so uploads from before the API inferred types
// become previewable.
//
// It rewrites only rows whose type is empty or generic, so a specific type is
// never overwritten, and it runs as raw SQL so updated_at keeps its value:
// bumping it would make every sync client re-fetch each fixed file and change
// the versioned thumbnail URL for no byte-level reason. The statement matches
// nothing once the rows are fixed, which makes running it at every boot cheap.
func backfillMimeTypes(db *gorm.DB) error {
	table := mimetype.Extensions()
	if len(table) == 0 {
		return nil
	}
	rows := make([]string, 0, len(table))
	args := make([]any, 0, len(table)*2+1)
	for ext, mediaType := range table {
		rows = append(rows, "(?, ?)")
		args = append(args, ext, mediaType)
	}
	args = append(args, mimetype.Generic)
	return db.Exec(
		`UPDATE files SET mime_type = known.media_type
		 FROM (VALUES `+strings.Join(rows, ", ")+`) AS known(ext, media_type)
		 WHERE lower(substring(files.name from '\.[^./]*$')) = known.ext
		   AND coalesce(files.mime_type, '') IN ('', ?)`, args...).Error
}

// ensureAdmin promotes the earliest account when no administrator exists, so a
// fresh instance is always manageable. It never runs once an admin is present.
func ensureAdmin(db *gorm.DB) error {
	var adminCount int64
	if err := db.Model(&User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
		return err
	}
	if adminCount > 0 {
		return nil
	}
	var firstUser User
	if err := db.Order("id asc").First(&firstUser).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return db.Model(&firstUser).Update("is_admin", true).Error
}
