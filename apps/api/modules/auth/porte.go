package auth

import (
	"context"
	"database/sql"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"

	"gorm.io/gorm"
)

// StoreColumns maps porte/kit onto Nuage's users table: the Registre picture
// lives in oidc_picture_url, and is_admin makes the first account the
// instance's administrator.
func StoreColumns() kit.Columns {
	return kit.Columns{Picture: "oidc_picture_url", Admin: "is_admin"}
}

const fillNewAccount = `UPDATE users SET color = $2, avatar_url = '', avatar_source = '',
	avatar_upload_path = '', password_hash = '', oidc_access_token = '', oidc_refresh_token = '',
	oidc_token_expiry = '0001-01-01 00:00:00+00', profile_synced_at = '0001-01-01 00:00:00+00'
	WHERE id = $1`

// UserStore resolves a Registre sign-in to a row in Nuage's users table. It is
// porte/kit's store with Nuage's two product rules added: every new account
// gets the least-used palette colour, and its creation emits user.created to
// Nook once the account is committed.
type UserStore struct {
	*kit.Store
	orm      *gorm.DB
	notifier *nook.Notifier
}

type createdKey struct{}

var _ porte.UserStore = (*UserStore)(nil)

// NewUserStore builds the store over orm's users table. The kit fills
// created_at itself; fill writes the colour and the empty values GORM used to
// write, so a scan into the plain string and time fields never meets a NULL.
func NewUserStore(ctx context.Context, orm *gorm.DB, notifier *nook.Notifier) (*UserStore, error) {
	sqlDB, err := orm.DB()
	if err != nil {
		return nil, err
	}
	users := &UserStore{orm: orm, notifier: notifier}
	users.Store, err = kit.NewStore(ctx, kit.Store{DB: sqlDB, Columns: StoreColumns(), OnCreate: users.fill})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserStore) fill(ctx context.Context, tx *sql.Tx, userID int64) error {
	color, err := usercolor.NextAvailable(ctx, s.orm)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fillNewAccount, userID, color); err != nil {
		return err
	}
	if created, ok := ctx.Value(createdKey{}).(*int64); ok {
		*created = userID
	}
	return nil
}

// UpsertFromOIDC applies porte/kit's linking rules and announces an account
// it created only after the transaction committed, so Nook never hears of an
// account that was rolled back.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	var created int64
	userID, err := s.Store.UpsertFromOIDC(context.WithValue(ctx, createdKey{}, &created), claims)
	if err == nil && created != 0 && s.notifier != nil {
		s.notifier.Notify(ctx, created, "user.created", nook.EventData{
			User: &nook.UserData{ID: created, Email: strings.ToLower(strings.TrimSpace(claims.Email))},
		})
	}
	return userID, err
}
