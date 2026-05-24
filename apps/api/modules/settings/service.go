package settings

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
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

func (s *Service) testNook(ctx context.Context) (bool, string, error) {
	var urlSetting schemas.Setting
	if err := s.orm.WithContext(ctx).Where("key = ?", "nook_webhook_url").First(&urlSetting).Error; err != nil {
		return false, "nook_webhook_url not configured", nil
	}
	if urlSetting.Value == "" {
		return false, "nook_webhook_url is empty", nil
	}

	var enabledSetting schemas.Setting
	if err := s.orm.WithContext(ctx).Where("key = ?", "nook_enabled").First(&enabledSetting).Error; err == nil {
		if enabledSetting.Value != "true" {
			return false, "nook is disabled", nil
		}
	}

	payload := map[string]string{
		"event": "ping",
		"time":  time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlSetting.Value, bytes.NewReader(body))
	if err != nil {
		return false, "failed to build request: " + err.Error(), nil
	}
	req.Header.Set("Content-Type", "application/json")

	var secretSetting schemas.Setting
	if err := s.orm.WithContext(ctx).Where("key = ?", "nook_webhook_secret").First(&secretSetting).Error; err == nil && secretSetting.Value != "" {
		mac := hmac.New(sha256.New, []byte(secretSetting.Value))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Nuage-Signature", sig)
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
