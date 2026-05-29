package auth

import (
	"context"
	stderrors "errors"
	"strconv"
	"strings"
	"time"

	"log/slog"
	"path/filepath"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/oidcavatar"
	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Service struct {
	orm        *gorm.DB
	notifier   *nook.Notifier
	storageDir string
	logger     *slog.Logger
	controller *Controller
}

func NewService(orm *gorm.DB, notifier *nook.Notifier, storageDir string, logger *slog.Logger) *Service {
	service := &Service{orm: orm, notifier: notifier, storageDir: storageDir, logger: logger}
	service.controller = newController(service)
	return service
}

func (service *Service) registerUser(context context.Context, email string, password string) (userID string, token string, err error) {
	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		return "", "", errors.Invalid("invalid password")
	}

	color, err := usercolor.NextAvailable(context, service.orm)
	if err != nil {
		return "", "", errors.Internal("failed to choose user color", err)
	}

	record := &schemas.User{
		Email:        email,
		Color:        color,
		PasswordHash: hash,
	}

	var userCount int64
	service.orm.WithContext(context).Model(&schemas.User{}).Count(&userCount)
	record.IsAdmin = userCount == 0

	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyErr(err) {
			return "", "", errors.Conflict("email already registered")
		}
		return "", "", errors.Internal("failed to create user", err)
	}

	if service.notifier != nil {
		service.notifier.Notify(context, record.ID, "user.created", nook.EventData{
			User: &nook.UserData{ID: record.ID, Email: record.Email},
		})
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}

	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) loginUser(context context.Context, email string, password string) (userID string, token string, err error) {
	var record schemas.User
	err = service.orm.WithContext(context).Where("email = ?", email).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", errors.Unauthorized("invalid credentials")
	}
	if err != nil {
		return "", "", errors.Internal("failed to read user", err)
	}
	if !authcrypto.VerifyPassword(password, record.PasswordHash) {
		return "", "", errors.Unauthorized("invalid credentials")
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}

	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) insertSession(context context.Context, token string, userID int64) error {
	record := &schemas.Session{
		Token:     authcrypto.HashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		return errors.Internal("failed to persist session", err)
	}
	return nil
}

func normalizeBearer(authorization string) string {
	value := strings.TrimSpace(authorization)
	if len(value) >= 7 && strings.EqualFold(value[:7], "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func (service *Service) authenticateRequest(context context.Context, authorization string) (string, *Data, error) {
	token := normalizeBearer(authorization)
	if token == "" {
		return "", nil, errors.Unauthorized("missing auth token")
	}

	hashed := authcrypto.HashToken(token)

	var out struct {
		UserID    int64
		Email     string
		ExpiresAt time.Time
	}
	err := service.orm.WithContext(context).
		Table("sessions s").
		Select("u.id as user_id, u.email as email, s.expires_at as expires_at").
		Joins("join users u on u.id = s.user_id").
		Where("s.token = ?", hashed).
		Scan(&out).Error
	if err != nil {
		return "", nil, errors.Internal("failed to validate auth token", err)
	}
	if out.UserID != 0 {
		if time.Now().After(out.ExpiresAt) {
			return "", nil, errors.Unauthorized("expired auth token")
		}
		return strconv.FormatInt(out.UserID, 10), &Data{Email: out.Email}, nil
	}

	var apiOut struct {
		UserID int64
		Email  string
	}
	err = service.orm.WithContext(context).
		Table("api_tokens t").
		Select("u.id as user_id, u.email as email").
		Joins("join users u on u.id = t.user_id").
		Where("t.token = ?", hashed).
		Scan(&apiOut).Error
	if err != nil {
		return "", nil, errors.Internal("failed to validate api token", err)
	}
	if apiOut.UserID == 0 {
		return "", nil, errors.Unauthorized("invalid auth token")
	}
	return strconv.FormatInt(apiOut.UserID, 10), &Data{Email: apiOut.Email}, nil
}

func (service *Service) Authenticate(context context.Context, authorization string) (string, any, error) {
	return service.authenticateRequest(context, authorization)
}

func (service *Service) deleteSession(ctx context.Context, authorization string) error {
	token := normalizeBearer(authorization)
	if token == "" {
		return errors.Unauthorized("missing auth token")
	}
	hashed := authcrypto.HashToken(token)
	result := service.orm.WithContext(ctx).Where("token = ?", hashed).Delete(&schemas.Session{})
	if result.Error != nil {
		return errors.Internal("failed to delete session", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("session not found")
	}
	return nil
}

func (service *Service) upsertOIDCUser(context context.Context, email string, profile oidcavatar.Profile, oauth2Token *oauth2.Token) (userID string, token string, err error) {
	var record schemas.User
	err = service.orm.WithContext(context).Where("email = ?", email).First(&record).Error
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", errors.Internal("failed to look up user", err)
	}
	isNew := stderrors.Is(err, gorm.ErrRecordNotFound)

	if isNew {
		color, colorErr := usercolor.NextAvailable(context, service.orm)
		if colorErr != nil {
			return "", "", errors.Internal("failed to choose user color", colorErr)
		}
		var userCount int64
		service.orm.WithContext(context).Model(&schemas.User{}).Count(&userCount)
		record = schemas.User{Email: email, Color: color, IsAdmin: userCount == 0}
		if displayName := profile.DisplayName(); displayName != "" {
			record.Name = displayName
		}
		if err := service.orm.WithContext(context).Create(&record).Error; err != nil {
			return "", "", errors.Internal("failed to create user", err)
		}
		if service.notifier != nil {
			service.notifier.Notify(context, record.ID, "user.created", nook.EventData{
				User: &nook.UserData{ID: record.ID, Email: record.Email},
			})
		}
		if profile.Picture != "" {
			relPath, fetchErr := oidcavatar.FetchAvatar(profile.Picture, service.storageDir, record.ID, service.logger)
			if fetchErr != nil {
				service.logger.Warn("failed to fetch OIDC avatar for new user", slog.Int64("user_id", record.ID), slog.Any("error", fetchErr))
			} else {
				record.AvatarURL = "/api/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
				record.AvatarSource = "oidc"
				record.OIDCPictureURL = profile.Picture
				service.orm.WithContext(context).Save(&record)
			}
		}
	} else {
		changed := false
		if displayName := profile.DisplayName(); displayName != "" {
			record.Name = displayName
			changed = true
		}
		if profile.Picture != "" {
			if profile.Picture != record.OIDCPictureURL {
				if record.AvatarSource != "upload" {
					if record.AvatarSource == "oidc" && record.AvatarURL != "" {
						oldRel := strings.TrimPrefix(record.AvatarURL, "/api/")
						oidcavatar.RemoveFile(service.storageDir, oldRel)
					}
					relPath, fetchErr := oidcavatar.FetchAvatar(profile.Picture, service.storageDir, record.ID, service.logger)
					if fetchErr != nil {
						service.logger.Warn("failed to fetch OIDC avatar", slog.Int64("user_id", record.ID), slog.Any("error", fetchErr))
					} else {
						record.AvatarURL = "/api/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
						record.AvatarSource = "oidc"
						changed = true
					}
				}
				record.OIDCPictureURL = profile.Picture
				changed = true
			}
		}
		if changed {
			service.orm.WithContext(context).Save(&record)
		}
	}

	if oauth2Token != nil {
		record.OIDCAccessToken = oauth2Token.AccessToken
		record.OIDCRefreshToken = oauth2Token.RefreshToken
		record.OIDCTokenExpiry = oauth2Token.Expiry
		service.orm.WithContext(context).Select("oidc_access_token", "oidc_refresh_token", "oidc_token_expiry").Save(&record)
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}
	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) SyncOIDCProfile(ctx context.Context, userID int64, provider *gooidc.Provider, oauth2Cfg *oauth2.Config) (bool, error) {
	var record schemas.User
	if err := service.orm.WithContext(ctx).First(&record, userID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.NotFound("user not found")
		}
		return false, errors.Internal("failed to load user", err)
	}

	if record.OIDCAccessToken == "" {
		return false, nil
	}

	if !record.ProfileSyncedAt.IsZero() && time.Since(record.ProfileSyncedAt) < 5*time.Minute {
		return false, nil
	}

	tok := &oauth2.Token{
		AccessToken:  record.OIDCAccessToken,
		RefreshToken: record.OIDCRefreshToken,
		Expiry:       record.OIDCTokenExpiry,
	}
	tokenSource := oauth2Cfg.TokenSource(ctx, tok)

	userInfo, err := provider.UserInfo(ctx, tokenSource)
	if err != nil {
		service.logger.Warn("OIDC profile sync failed, clearing tokens", slog.Int64("user_id", userID), slog.Any("error", err))
		service.orm.WithContext(ctx).Model(&record).Updates(map[string]any{
			"oidc_access_token":  "",
			"oidc_refresh_token": "",
		})
		return false, nil
	}

	var claims struct {
		Name             string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName        string `json:"given_name"`
		FamilyName       string `json:"family_name"`
		Picture          string `json:"picture"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		return false, errors.Internal("failed to parse userinfo claims", err)
	}

	profile := oidcavatar.Profile{
		Name:             claims.Name,
		PreferredUsername: claims.PreferredUsername,
		GivenName:        claims.GivenName,
		FamilyName:       claims.FamilyName,
		Picture:          claims.Picture,
	}

	changed := false
	if displayName := profile.DisplayName(); displayName != "" && displayName != record.Name {
		record.Name = displayName
		changed = true
	}
	if profile.Picture != "" && profile.Picture != record.OIDCPictureURL {
		if record.AvatarSource != "upload" {
			if record.AvatarSource == "oidc" && record.AvatarURL != "" {
				oldRel := strings.TrimPrefix(record.AvatarURL, "/api/")
				oidcavatar.RemoveFile(service.storageDir, oldRel)
			}
			relPath, fetchErr := oidcavatar.FetchAvatar(profile.Picture, service.storageDir, record.ID, service.logger)
			if fetchErr != nil {
				service.logger.Warn("failed to fetch OIDC avatar during sync", slog.Int64("user_id", record.ID), slog.Any("error", fetchErr))
			} else {
				record.AvatarURL = "/api/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
				record.AvatarSource = "oidc"
				changed = true
			}
		}
		record.OIDCPictureURL = profile.Picture
		changed = true
	}

	refreshedToken, err := tokenSource.Token()
	if err == nil && refreshedToken.AccessToken != tok.AccessToken {
		record.OIDCAccessToken = refreshedToken.AccessToken
		record.OIDCRefreshToken = refreshedToken.RefreshToken
		record.OIDCTokenExpiry = refreshedToken.Expiry
		changed = true
	}

	record.ProfileSyncedAt = time.Now()
	service.orm.WithContext(ctx).Save(&record)

	return changed, nil
}

func isDuplicateKeyErr(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "SQLSTATE 23505")
}
