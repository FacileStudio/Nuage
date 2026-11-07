package auth

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/oidcavatar"
	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// UserStore resolves a login to a row in Nuage's own users table.
//
// It is porte's escape hatch, taken deliberately. porte/pg ships a porte_users
// table and Nuage could have moved onto it, but files, folders, shares, quotas
// and space memberships all hang off users(id), and the row carries is_admin
// and an assigned display colour.
//
// Those last two are the reason porte asks the app to own this method rather
// than owning the write itself: choosing a colour requires knowing which ones
// the other users already hold, and deciding that the first account ever
// created is an admin is a product rule. A library that made either decision
// would be routed around by the second app that adopted it.
type UserStore struct {
	orm      *gorm.DB
	notifier *nook.Notifier
}

// NewUserStore builds a UserStore backed by the given database connection,
// notifying through notifier when accounts are created.
func NewUserStore(orm *gorm.DB, notifier *nook.Notifier) *UserStore {
	return &UserStore{orm: orm, notifier: notifier}
}

var _ porte.UserStore = (*UserStore)(nil)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a verified
// email, and creates the account when neither finds anything.
//
// porte has already done the matching it can: it looks up the identity row and
// only calls this when it needs a user id. The email fallback here is what
// links a pre-existing account to a subject signing in for the first time. It
// is refused for an unverified address because matching on one is an account
// takeover primitive, and refused when the account already holds another
// subject of the same provider, so a recycled address is not handed to a
// newcomer (porte v0.10.0's linking rule, which an app with its own users table
// has to apply itself).
//
// The identity provider is the source of truth for the profile, but an
// absent claim asserts nothing: a provider that stops sending a name
// should not blank it here. oidc_picture_url is written unconditionally
// because an emptied one means "there is no SSO photo", which is what
// makes the uploaded avatar show again. The email is the provider's too: it
// refreshes at every sign-in, which is why the account cannot edit it locally.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return 0, errors.Invalid("the identity provider returned no email")
	}

	user, err := s.resolve(ctx, claims, email)
	if err != nil {
		return 0, err
	}

	updates := map[string]any{"email": email, "oidc_picture_url": oidcavatar.PhotoURL(claims.Picture)}
	if displayName := claims.DisplayName(); displayName != "" {
		updates["name"] = displayName
	}
	if claims.Locale != "" {
		updates["locale"] = claims.Locale
	}
	if err := s.orm.WithContext(ctx).Model(&schemas.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return 0, errors.Internal("failed to update the account", err)
	}
	return user.ID, nil
}

// resolve finds the account a sign-in belongs to: by (provider, subject), then
// by a verified email, and otherwise a new one.
func (s *UserStore) resolve(ctx context.Context, claims porte.Claims, email string) (*schemas.User, error) {
	var user schemas.User
	err := s.orm.WithContext(ctx).
		Joins("JOIN porte_identities i ON i.user_id = users.id AND i.provider = ? AND i.subject = ?", claims.Provider, claims.Subject).
		First(&user).Error
	switch {
	case err == nil:
		return &user, nil
	case !stderrors.Is(err, gorm.ErrRecordNotFound):
		return nil, errors.Internal("failed to resolve the identity", err)
	}

	err = s.orm.WithContext(ctx).Where("email = ?", email).First(&user).Error
	switch {
	case err == nil:
		return &user, s.checkLinkable(ctx, user.ID, claims)
	case stderrors.Is(err, gorm.ErrRecordNotFound):
		return s.create(ctx, email, claims.DisplayName(), claims.Locale)
	}
	return nil, errors.Internal("failed to look up the account", err)
}

// checkLinkable refuses to attach a sign-in to an existing account by email
// unless the address is verified and the account holds no other subject of the
// same provider.
func (s *UserStore) checkLinkable(ctx context.Context, userID int64, claims porte.Claims) error {
	if !claims.EmailVerified {
		return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
	}
	var held int64
	err := s.orm.WithContext(ctx).Table("porte_identities").
		Where("user_id = ? AND provider = ?", userID, claims.Provider).
		Count(&held).Error
	if err != nil {
		return errors.Internal("failed to check the account's identities", err)
	}
	if held > 0 {
		return errors.Conflict("an account with this email is already linked to another identity")
	}
	return nil
}

// create is the one place a Nuage account comes into existence, whichever way
// the human arrived. Keeping it to one function is what stops the colour, the
// first-admin rule and the Nook event from being applied on one path and
// forgotten on another.
//
// The first account ever created runs the instance. Counting inside the
// same call that inserts is a race in theory; in practice the first
// account is created by one human on a fresh deployment, and a unique
// index on email means the loser of any real race is refused anyway.
func (s *UserStore) create(ctx context.Context, email, name, locale string) (*schemas.User, error) {
	color, err := usercolor.NextAvailable(ctx, s.orm)
	if err != nil {
		return nil, errors.Internal("failed to choose a user colour", err)
	}

	var existing int64
	if err := s.orm.WithContext(ctx).Model(&schemas.User{}).Count(&existing).Error; err != nil {
		return nil, errors.Internal("failed to count users", err)
	}

	user := schemas.User{Email: email, Name: name, Locale: locale, Color: color, IsAdmin: existing == 0}
	if err := s.orm.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, errors.Internal("failed to create the account", err)
	}
	if s.notifier != nil {
		s.notifier.Notify(ctx, user.ID, "user.created", nook.EventData{
			User: &nook.UserData{ID: user.ID, Email: user.Email},
		})
	}
	return &user, nil
}
