package auth

import (
	"net/http"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) register(w http.ResponseWriter, r *http.Request, req *RegisterRequest) (*AuthResponse, error) {
	context := r.Context()
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !isValidEmail(email) {
		return nil, errors.Invalid("invalid email")
	}
	if len(req.Password) < 12 {
		return nil, errors.Invalid("password must be at least 12 characters")
	}

	userID, token, err := controller.service.Register(context, w, r, email, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{UserID: userID, Token: token}, nil
}

func (controller *Controller) login(w http.ResponseWriter, r *http.Request, req *LoginRequest) (*AuthResponse, error) {
	context := r.Context()
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, errors.Invalid("email and password required")
	}

	userID, token, err := controller.service.Login(context, w, r, email, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{UserID: userID, Token: token}, nil
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
