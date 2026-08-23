package users

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
)

// Controller is the HTTP controller for the users module.
type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) list(context context.Context) (*ListResponse, error) {
	if _, ok := authcontext.IdentityFromContext(context); !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	users, err := controller.service.listUsers(context)
	if err != nil {
		return nil, err
	}

	return &ListResponse{Users: users}, nil
}

func (controller *Controller) get(context context.Context, userID string) (*MeResponse, error) {
	user, err := controller.service.getUser(context, userID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{User: *user}, nil
}

func (controller *Controller) me(context context.Context) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	user, err := controller.service.getUser(context, identity.UserID)
	if err != nil {
		return nil, err
	}

	if user.Email == "" {
		user.Email = identity.Email
	}

	return &MeResponse{User: *user}, nil
}

// updateMe applies a profile edit, and is where the password path lives rather
// than in the service: porte rotates the caller's session cookie itself, so
// the write needs the ResponseWriter and the request that only a handler holds.
//
// Moving the address the account signs in with is confirmed with the current
// password, the same as replacing one, because a borrowed session must not be
// enough to do it. That confirmation is asked for here only when the body
// carries no password: when it carries one, porte's ChangePassword confirms it
// a few lines below and asking twice would hash argon2 twice for one request.
// An account with no password has nothing to confirm either way, which is the
// branch SetPassword serves.
func (controller *Controller) updateMe(context context.Context, w http.ResponseWriter, request *http.Request, req *UpdateRequest) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if len(trimmed) > 80 {
			return nil, errors.Invalid("name must be at most 80 characters")
		}
		name = &trimmed
	}

	var email *string
	if req.Email != nil {
		normalized := strings.TrimSpace(strings.ToLower(*req.Email))
		if normalized == "" || !strings.Contains(normalized, "@") {
			return nil, errors.Invalid("invalid email")
		}
		email = &normalized
	}

	var password *string
	if req.Password != nil {
		if len(*req.Password) < 12 {
			return nil, errors.Invalid("password must be at least 12 characters")
		}
		password = req.Password
	}

	if email != nil && password == nil {
		if req.CurrentPassword == nil || strings.TrimSpace(*req.CurrentPassword) == "" {
			return nil, errors.Invalid("current_password is required to change the email")
		}
		if err := controller.service.verifyPassword(context, identity.UserID, *req.CurrentPassword); err != nil {
			return nil, err
		}
	}

	var color *string
	if req.Color != nil {
		normalized, ok := usercolor.Normalize(*req.Color)
		if !ok {
			return nil, errors.Invalid("color must be one of: AD9EF0, F09ED6, EE7E89, EEB47E, A9EE7E, 7EEEDB")
		}
		color = &normalized
	}

	if name == nil && email == nil && password == nil && color == nil {
		return nil, errors.Invalid("at least one field must be provided")
	}

	var rotated string
	if password != nil {
		token, err := controller.changePassword(context, w, request, identity.UserID, req, *password)
		if err != nil {
			return nil, err
		}
		rotated = token
	}

	user, err := controller.service.updateUser(context, identity.UserID, name, email, color)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: *user, Token: rotated}, nil
}

// changePassword picks between porte's two password writes and returns the
// caller's replacement token when there is one.
//
// They are two calls rather than one because only one of them is safe to make
// on a session alone: SetPassword gives a first password to an account that
// has none, and porte refuses it with ErrPasswordSet once there is one.
// Replacing a password goes through ChangePassword, which confirms the current
// one — OWASP ASVS puts that at L1 (v4 2.1.6, v5 6.2.3) — then ends the
// account's other logins and rotates this caller's session through w.
//
// ErrPasswordSet is answered as a 400 naming the field rather than porte's
// 409, because the caller left something out rather than losing a race. A
// blank current password counts as none given, so it reaches the same answer
// instead of "invalid credentials".
func (controller *Controller) changePassword(context context.Context, w http.ResponseWriter, request *http.Request, userID string, req *UpdateRequest, password string) (string, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return "", errors.Internal("failed to parse user id", err)
	}

	current := ""
	if req.CurrentPassword != nil {
		current = strings.TrimSpace(*req.CurrentPassword)
	}
	if current != "" {
		token, _, err := controller.service.tokens.ChangePassword(context, w, request, id, current, password)
		return token, err
	}

	err = controller.service.tokens.SetPassword(context, id, password)
	if stderrors.Is(err, porte.ErrPasswordSet) {
		return "", errors.Invalid("current_password is required to change your password")
	}
	return "", err
}

func (controller *Controller) deleteAvatar(context context.Context) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	user, err := controller.service.clearAvatar(context, identity.UserID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{User: *user}, nil
}

func (controller *Controller) uploadAvatar(context context.Context, request *http.Request) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	if err := request.ParseMultipartForm(5 << 20); err != nil {
		return nil, errors.TooLarge("avatar file is too large")
	}

	file, _, err := request.FormFile("avatar")
	if err != nil {
		return nil, errors.Invalid("avatar file is required")
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return nil, errors.Internal("failed to read avatar file", err)
	}

	contentType := http.DetectContentType(header[:n])
	user, err := controller.service.storeAvatar(context, identity.UserID, io.MultiReader(bytes.NewReader(header[:n]), file), contentType)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: *user}, nil
}

func (controller *Controller) getApiToken(context context.Context) (*ApiTokenListResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	records, err := controller.service.getApiTokens(context, identity.UserID)
	if err != nil {
		return nil, err
	}
	tokens := make([]ApiTokenResponse, 0, len(records))
	for _, r := range records {
		tokens = append(tokens, ApiTokenResponse{
			ID:        r.ID,
			Name:      r.Label,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return &ApiTokenListResponse{Tokens: tokens}, nil
}

func (controller *Controller) createApiToken(context context.Context, req *CreateApiTokenRequest) (*ApiTokenResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "CLI"
	}
	rawToken, record, err := controller.service.createApiToken(context, identity.UserID, name)
	if err != nil {
		return nil, err
	}
	return &ApiTokenResponse{
		ID:        record.ID,
		Token:     rawToken,
		Name:      record.Label,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (controller *Controller) deleteApiToken(context context.Context, tokenID string) error {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return errors.Unauthorized("missing auth")
	}
	id, err := strconv.ParseInt(tokenID, 10, 64)
	if err != nil {
		return errors.Invalid("invalid token id")
	}
	return controller.service.deleteApiToken(context, identity.UserID, id)
}
