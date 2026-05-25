package settings

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var allowedKeys = map[string]bool{
	"nook_webhook_url":    true,
	"nook_webhook_secret": true,
	"nook_enabled":        true,
	"instance_name":       true,
}

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) listSettings(ctx context.Context) ([]schemas.Setting, error) {
	var records []schemas.Setting
	if err := s.orm.WithContext(ctx).Order("key asc").Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list settings", err)
	}
	return records, nil
}

func (s *Service) updateSettings(ctx context.Context, values map[string]string) ([]schemas.Setting, error) {
	for key := range values {
		if !allowedKeys[key] {
			return nil, errors.Invalid("unknown setting key: " + key)
		}
	}

	now := time.Now()
	for key, value := range values {
		record := schemas.Setting{
			Key:       key,
			Value:     value,
			UpdatedAt: now,
		}
		if err := s.orm.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&record).Error; err != nil {
			return nil, errors.Internal("failed to update setting", err)
		}
	}

	return s.listSettings(ctx)
}

func (s *Service) testNook(ctx context.Context, input TestNookRequest) (bool, string, error) {
	webhookURL := input.URL
	secret := input.Secret

	if webhookURL == "" {
		return false, "webhook URL is empty", nil
	}

	if !input.Enabled {
		return false, "nook is disabled", nil
	}

	parsed, err := url.Parse(webhookURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return false, "invalid webhook URL: must start with http:// or https://", nil
	}

	payload := map[string]string{
		"event": "ping",
		"time":  time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return false, "failed to build request: " + err.Error(), nil
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Nuage-Signature-256", sig)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "request failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, "ping successful", nil
	}
	return false, "nook responded with status " + resp.Status, nil
}
