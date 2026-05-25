package nook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Actor struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type EventData struct {
	File   *FileData   `json:"file,omitempty"`
	Folder *FolderData `json:"folder,omitempty"`
	Share  *ShareData  `json:"share,omitempty"`
}

type FileData struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type FolderData struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ShareData struct {
	ID         int64  `json:"id"`
	Permission string `json:"permission"`
}

type Event struct {
	EventType    string    `json:"event_type"`
	OccurredAt   string    `json:"occurred_at"`
	InstanceName string    `json:"instance_name"`
	Actor        Actor     `json:"actor"`
	Data         EventData `json:"data"`
}

type Notifier struct {
	orm *gorm.DB
}

func NewNotifier(orm *gorm.DB) *Notifier {
	return &Notifier{orm: orm}
}

func (n *Notifier) Notify(ctx context.Context, actorID int64, eventType string, data EventData) {
	go n.dispatch(actorID, eventType, data)
}

func (n *Notifier) dispatch(actorID int64, eventType string, data EventData) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var urlSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_webhook_url").First(&urlSetting).Error; err != nil || urlSetting.Value == "" {
		return
	}

	var enabledSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_enabled").First(&enabledSetting).Error; err == nil {
		if enabledSetting.Value != "true" {
			return
		}
	}

	var user schemas.User
	if err := n.orm.WithContext(ctx).Where("id = ?", actorID).First(&user).Error; err != nil {
		slog.Warn("nook: failed to resolve actor", slog.Any("error", err))
		return
	}

	var instanceName string
	var nameSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "instance_name").First(&nameSetting).Error; err == nil {
		instanceName = nameSetting.Value
	}

	event := Event{
		EventType:    eventType,
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
		InstanceName: instanceName,
		Actor: Actor{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
		Data: data,
	}

	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("nook: failed to marshal event", slog.Any("error", err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlSetting.Value, bytes.NewReader(body))
	if err != nil {
		slog.Warn("nook: failed to build request", slog.Any("error", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	var secretSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_webhook_secret").First(&secretSetting).Error; err == nil && secretSetting.Value != "" {
		mac := hmac.New(sha256.New, []byte(secretSetting.Value))
		mac.Write(body)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Nuage-Signature-256", sig)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("nook: webhook delivery failed", slog.Any("error", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slog.Warn("nook: webhook returned non-success", slog.Int("status", resp.StatusCode))
	}
}
