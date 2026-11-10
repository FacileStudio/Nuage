package spaces

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"

	"gorm.io/gorm"
)

// resolveMember returns the local user id for a snapshot member, or 0 to skip them.
//
// It tries, in order: the porte identity (issuer, sub); a local user with the same email,
// which gains that identity; a new user with the identity already attached, so their first
// sign-in lands on the row that already holds their memberships. An email match whose user
// already has a different subject under the issuer is skipped and logged rather than given
// a second one.
func (r *Registre) resolveMember(ctx context.Context, tx *gorm.DB, member directory.Member) (int64, error) {
	if member.Subject == "" {
		return 0, nil
	}
	userID, err := r.userBySubject(tx, member.Subject)
	if err != nil || userID != 0 {
		return userID, err
	}
	email := strings.ToLower(strings.TrimSpace(member.Email))
	if email == "" {
		return 0, nil
	}
	userID, err = r.userByEmail(tx, email, member.Subject)
	if err != nil || userID != 0 {
		return userID, err
	}
	if userID, err = provisionUser(ctx, tx, email, member); err != nil {
		return 0, err
	}
	return userID, r.attachIdentity(tx, userID, member.Subject)
}

// userBySubject finds the user holding the identity (issuer, subject), or 0.
func (r *Registre) userBySubject(tx *gorm.DB, subject string) (int64, error) {
	var ids []int64
	err := tx.Table("porte_identities").Where("provider = ? AND subject = ?", r.issuer, subject).
		Limit(1).Pluck("user_id", &ids).Error
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	return ids[0], nil
}

// userByEmail links the local user with this email to the subject and returns them, or 0
// when no user has the address or the one who does already holds another subject.
func (r *Registre) userByEmail(tx *gorm.DB, email, subject string) (int64, error) {
	var user schemas.User
	err := tx.Select("id").Where("lower(email) = ?", email).Order("id asc").First(&user).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var held int64
	err = tx.Table("porte_identities").Where("provider = ? AND user_id = ?", r.issuer, user.ID).Count(&held).Error
	if err != nil {
		return 0, err
	}
	if held > 0 {
		r.logger.Warn("directory: skipped a member whose email belongs to another registre identity",
			slog.Int64("user_id", user.ID))
		return 0, nil
	}
	return user.ID, r.attachIdentity(tx, user.ID, subject)
}

// attachIdentity records the user's identity under the issuer, with no tokens: porte fills
// those in at the first sign-in.
func (r *Registre) attachIdentity(tx *gorm.DB, userID int64, subject string) error {
	return tx.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)
		ON CONFLICT (provider, subject) DO NOTHING`, userID, r.issuer, subject).Error
}

// provisionUser creates the account a snapshot member will sign in to, with Registre's name
// and picture and the least-used colour, and no password. It is never an instance admin.
func provisionUser(ctx context.Context, tx *gorm.DB, email string, member directory.Member) (int64, error) {
	color, err := usercolor.NextAvailable(ctx, tx)
	if err != nil {
		return 0, err
	}
	user := schemas.User{
		Email:          email,
		Name:           strings.TrimSpace(member.Name),
		Color:          color,
		OIDCPictureURL: kit.PictureURL(member.Picture),
	}
	if err := tx.Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}
