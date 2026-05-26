package auth

import (
	"context"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) register(context context.Context, req *RegisterRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(email) {
		return nil, errors.Invalid("invalid email")
	}
	if len(req.Password) < 12 {
		return nil, errors.Invalid("password must be at least 12 characters")
	}

	userID, token, err := controller.service.registerUser(context, email, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{UserID: userID, Token: token}, nil
}

func (controller *Controller) login(context context.Context, req *LoginRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, errors.Invalid("email and password required")
	}

	userID, token, err := controller.service.loginUser(context, email, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{UserID: userID, Token: token}, nil
}

func (controller *Controller) authenticate(context context.Context, authorization string) (string, *Data, error) {
	return controller.service.authenticateRequest(context, authorization)
}

func isValidEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	if local == "" || len(local) > 64 {
		return false
	}
	if domain == "" || !strings.Contains(domain, ".") {
		return false
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	return true
}
