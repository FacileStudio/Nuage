package users

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service implements user management, profiles, avatars and API tokens.
type Service struct {
	orm        *gorm.DB
	storageDir string
	tokens     Auth
	controller *Controller
}

// Auth is the auth service, narrowed to what this module needs of it.
type Auth interface {
	Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error)
	Sessions() *session.Manager
	SetPassword(ctx context.Context, userID int64, email, password string) error
	VerifyPassword(ctx context.Context, email, password string) (int64, error)
	RevokeBrowserSessions(ctx context.Context, userID int64) error
}

// NewService builds a users Service over the given database connection,
// storage directory and token-issuing auth dependency.
func NewService(orm *gorm.DB, storageDir string, tokens Auth) *Service {
	service := &Service{orm: orm, storageDir: storageDir, tokens: tokens}
	service.controller = newController(service)
	return service
}

func (service *Service) getUser(context context.Context, userID string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}
	if err := service.ensureUserColor(context, &record); err != nil {
		return nil, err
	}

	return mapUser(record), nil
}

func (service *Service) listUsers(context context.Context) ([]User, error) {
	if err := usercolor.BackfillMissing(context, service.orm); err != nil {
		return nil, errors.Internal("failed to backfill user colors", err)
	}

	var records []schemas.User
	if err := service.orm.WithContext(context).Order("name asc, email asc, id asc").Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list users", err)
	}

	users := make([]User, 0, len(records))
	for _, record := range records {
		users = append(users, *mapUser(record))
	}

	return users, nil
}

// verifyPassword confirms the caller knows the account's current password, so a
// stolen session token alone cannot be escalated into a permanent takeover by
// changing the login credentials.
//
// porte reads porte_identities, not users.password_hash. That column
// still holds the hash the migration copied out of it, so comparing
// against it here would keep answering yes to the *old* password after
// a change — the same silent divergence, read from the other side.
func (service *Service) verifyPassword(context context.Context, userID string, candidate string) error {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("user not found")
		}
		return errors.Internal("failed to read user", err)
	}

	if _, err := service.tokens.VerifyPassword(context, record.Email, candidate); err != nil {
		return errors.Unauthorized("current password is incorrect")
	}
	return nil
}

// updateUser applies a partial profile update for a user, re-keying porte's
// local identity when the email changes.
//
// The password is porte's, not a column on this row. Writing
// users.password_hash would look like it worked and change nothing:
// porte reads the identity table, so the old password would keep
// signing in and the new one would never work.
//
// The address is half of porte's key for a local identity, so changing
// it without moving that key breaks the login on the next profile edit.
// Re-key first, then set the password on the address the account will
// actually have.
//
// Changing a password signs the other browsers out. It deliberately
// spares named API tokens: before porte they lived in their own table
// and a DELETE on sessions never touched them, so taking them now would
// break somebody's script on the day they rotate their password.
func (service *Service) updateUser(context context.Context, userID string, name *string, email *string, password *string, color *string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	updates := map[string]any{}
	if name != nil {
		updates["name"] = *name
	}

	if email != nil || password != nil {
		var current schemas.User
		if err := service.orm.WithContext(context).Select("email").First(&current, id).Error; err != nil {
			return nil, errors.Internal("failed to read the account", err)
		}
		address := current.Email
		if email != nil {
			address = *email
			updates["email"] = address
			if !strings.EqualFold(address, current.Email) {
				if err := service.orm.WithContext(context).Exec(
					`UPDATE porte_identities SET subject = ? WHERE provider = 'local' AND subject = ?`,
					strings.ToLower(strings.TrimSpace(address)),
					strings.ToLower(strings.TrimSpace(current.Email)),
				).Error; err != nil {
					return nil, errors.Internal("failed to move the password to the new address", err)
				}
			}
		}
		if password != nil {
			if err := service.tokens.SetPassword(context, id, address, *password); err != nil {
				return nil, err
			}
		}
	}
	if color != nil {
		updates["color"] = *color
	}

	result := service.orm.WithContext(context).
		Model(&schemas.User{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		if stderrors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("email already registered")
		}
		return nil, errors.Internal("failed to update user", result.Error)
	}
	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}
	if err := service.ensureUserColor(context, &record); err != nil {
		return nil, err
	}

	if password != nil {
		if err := service.tokens.RevokeBrowserSessions(context, id); err != nil {
			return nil, err
		}
	}

	return mapUser(record), nil
}

// storeAvatar persists an uploaded avatar for a user, writing the file
// under the avatars directory and recording its path on the account.
//
// Uploading is the fallback for people the IdP has no photo for, so a
// photo in Porte makes this endpoint unavailable rather than merely
// outranked. Accepting the file and then never showing it is the worse
// failure: the user sees a success and no change.
func (service *Service) storeAvatar(context context.Context, userID string, reader io.Reader, contentType string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	if record.OIDCPictureURL != "" {
		return nil, errors.Invalid("your photo is managed in single sign-on — change it there")
	}

	filename, absolutePath, err := service.persistAvatarFile(id, reader, contentType)
	if err != nil {
		return nil, err
	}

	oldFilename := record.AvatarUploadPath
	record.AvatarUploadPath = filename

	if err := service.orm.WithContext(context).Save(&record).Error; err != nil {
		_ = os.Remove(absolutePath)
		return nil, errors.Internal("failed to save avatar", err)
	}

	service.removeAvatarFile(oldFilename)

	if err := service.ensureUserColor(context, &record); err != nil {
		return nil, err
	}

	return mapUser(record), nil
}

// clearAvatar removes a user's uploaded avatar, leaving any Porte-managed
// photo untouched.
//
// Only the upload is the user's to clear. The Porte photo is not deleted
// from here — it is not ours, and the next profile sync would bring it
// straight back.
func (service *Service) clearAvatar(context context.Context, userID string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	oldFilename := record.AvatarUploadPath
	record.AvatarUploadPath = ""
	if err := service.orm.WithContext(context).Save(&record).Error; err != nil {
		return nil, errors.Internal("failed to clear avatar", err)
	}

	service.removeAvatarFile(oldFilename)

	if err := service.ensureUserColor(context, &record); err != nil {
		return nil, err
	}

	return mapUser(record), nil
}

func (service *Service) persistAvatarFile(userID int64, reader io.Reader, contentType string) (string, string, error) {
	extension, ok := avatarExtension(contentType)
	if !ok {
		return "", "", errors.Invalid("avatar must be a PNG, JPEG, GIF, or WebP image")
	}

	filename := fmt.Sprintf("user-%d-%d%s", userID, time.Now().UnixNano(), extension)
	absolutePath := service.avatarFilePath(filename)

	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", "", errors.Internal("failed to prepare avatar storage", err)
	}

	file, err := os.Create(absolutePath)
	if err != nil {
		return "", "", errors.Internal("failed to create avatar file", err)
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return "", "", errors.Internal("failed to write avatar file", err)
	}
	if err := file.Close(); err != nil {
		return "", "", errors.Internal("failed to finalize avatar file", err)
	}

	return filename, absolutePath, nil
}

// avatarFilePath resolves a stored avatar filename inside the avatars directory. Base()
// is what keeps a crafted stored value from escaping it — the column holds a bare
// filename, never a path.
func (service *Service) avatarFilePath(filename string) string {
	return filepath.Join(service.storageDir, "avatars", filepath.Base(filename))
}

func (service *Service) removeAvatarFile(filename string) {
	if filename == "" {
		return
	}
	_ = os.Remove(service.avatarFilePath(filename))
}

func (service *Service) ensureUserColor(context context.Context, record *schemas.User) error {
	color, ok := usercolor.Normalize(record.Color)
	if ok && record.Color == color {
		return nil
	}

	color, err := usercolor.EnsureForUser(context, service.orm, record.ID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("user not found")
		}
		return errors.Internal("failed to assign user color", err)
	}

	record.Color = color
	return nil
}

func mapUser(record schemas.User) *User {
	return &User{
		ID:           strconv.FormatInt(record.ID, 10),
		Email:        record.Email,
		Name:         record.Name,
		AvatarURL:    record.Avatar(),
		AvatarSource: record.AvatarOrigin(),
		Color:        record.Color,
		CreatedAt:    record.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// A named API token is a porte session with a label on it and no expiry, so
// there is no second credential table and no second branch in the auth path.
// porte hands out ids, which is what the delete route already addressed them
// by.
func (service *Service) createApiToken(ctx context.Context, userID string, name string) (string, *porte.Session, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return "", nil, errors.Internal("failed to parse user id", err)
	}
	rawToken, issued, err := service.tokens.Issue(ctx, id, name)
	if err != nil {
		return "", nil, err
	}
	return rawToken, &issued, nil
}

// getApiTokens lists the labelled sessions only. The unlabelled ones are
// browser logins, and showing somebody their own laptop as an API token — with
// a revoke button next to it — is not the same feature.
func (service *Service) getApiTokens(ctx context.Context, userID string) ([]porte.Session, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}
	held, err := service.tokens.Sessions().List(ctx, id)
	if err != nil {
		return nil, errors.Internal("failed to read api tokens", err)
	}
	named := make([]porte.Session, 0, len(held))
	for _, candidate := range held {
		if candidate.Label != "" {
			named = append(named, candidate)
		}
	}
	return named, nil
}

// deleteApiToken revokes one, and refuses to revoke a browser login through
// the API-token route: porte's Revoke already scopes to the caller's own id,
// and the label check keeps this endpoint to the thing it names.
func (service *Service) deleteApiToken(ctx context.Context, userID string, tokenID int64) error {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return errors.Internal("failed to parse user id", err)
	}
	held, err := service.tokens.Sessions().List(ctx, id)
	if err != nil {
		return errors.Internal("failed to read api tokens", err)
	}
	for _, candidate := range held {
		if candidate.ID == tokenID && candidate.Label != "" {
			return service.tokens.Sessions().Revoke(ctx, id, tokenID)
		}
	}
	return errors.NotFound("token not found")
}

func avatarExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}
