package nook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
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
	User   *UserData   `json:"user,omitempty"`
	Quota  *QuotaData  `json:"quota,omitempty"`
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
	Token      string `json:"token"`
	Permission string `json:"permission"`
	FileID     *int64 `json:"file_id,omitempty"`
	FolderID   *int64 `json:"folder_id,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	FolderName string `json:"folder_name,omitempty"`
}

type UserData struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type QuotaData struct {
	UserID       int64 `json:"user_id"`
	StorageUsed  int64 `json:"storage_used"`
	StorageLimit int64 `json:"storage_limit"`
}

type Event struct {
	EventType    string    `json:"event_type"`
	OccurredAt   string    `json:"occurred_at"`
	InstanceName string    `json:"instance_name"`
	Actor        Actor     `json:"actor"`
	Data         EventData `json:"data"`
}

var retryDelays = []time.Duration{10 * time.Second, 60 * time.Second, 300 * time.Second}

type Notifier struct {
	orm    *gorm.DB
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	client *http.Client
}

func NewNotifier(orm *gorm.DB) *Notifier {
	ctx, cancel := context.WithCancel(context.Background())
	return &Notifier{
		orm:    orm,
		ctx:    ctx,
		cancel: cancel,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *Notifier) Start() {
	n.wg.Add(1)
	go n.worker()
}

func (n *Notifier) Stop() {
	n.cancel()
	n.wg.Wait()
}

func (n *Notifier) Notify(ctx context.Context, actorID int64, eventType string, data EventData) {
	go n.enqueue(actorID, eventType, data)
}

func (n *Notifier) enqueue(actorID int64, eventType string, data EventData) {
	ctx, cancel := context.WithTimeout(n.ctx, 15*time.Second)
	defer cancel()

	if !n.isEnabled(ctx) {
		return
	}

	if !n.matchesFilter(ctx, eventType) {
		return
	}

	payload, err := n.buildPayload(ctx, actorID, eventType, data)
	if err != nil {
		slog.Warn("nook: failed to build payload", slog.Any("error", err))
		return
	}

	now := time.Now()
	delivery := schemas.NookDelivery{
		EventType:   eventType,
		Payload:     string(payload),
		Status:      "pending",
		Attempts:    0,
		NextRetryAt: &now,
		CreatedAt:   now,
	}
	if err := n.orm.WithContext(ctx).Create(&delivery).Error; err != nil {
		slog.Warn("nook: failed to enqueue delivery", slog.Any("error", err))
	}
}

func (n *Notifier) isEnabled(ctx context.Context) bool {
	var urlSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_webhook_url").First(&urlSetting).Error; err != nil || urlSetting.Value == "" {
		return false
	}
	var enabledSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_enabled").First(&enabledSetting).Error; err == nil {
		if enabledSetting.Value != "true" {
			return false
		}
	}
	return true
}

func (n *Notifier) matchesFilter(ctx context.Context, eventType string) bool {
	var setting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_event_types").First(&setting).Error; err != nil || setting.Value == "" {
		return true
	}
	var types []string
	if err := json.Unmarshal([]byte(setting.Value), &types); err != nil {
		return true
	}
	if len(types) == 0 {
		return true
	}
	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

func (n *Notifier) buildPayload(ctx context.Context, actorID int64, eventType string, data EventData) ([]byte, error) {
	var user schemas.User
	if err := n.orm.WithContext(ctx).Where("id = ?", actorID).First(&user).Error; err != nil {
		return nil, err
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
		Actor:        Actor{ID: user.ID, Email: user.Email, Name: user.Name},
		Data:         data,
	}

	return json.Marshal(event)
}

func (n *Notifier) worker() {
	defer n.wg.Done()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.processQueue()
		}
	}
}

func (n *Notifier) processQueue() {
	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	var deliveries []schemas.NookDelivery
	now := time.Now()
	if err := n.orm.WithContext(ctx).
		Where("status = ? AND next_retry_at <= ?", "pending", now).
		Order("created_at asc").
		Limit(10).
		Find(&deliveries).Error; err != nil {
		return
	}
	if len(deliveries) == 0 {
		return
	}

	webhookURL, secret, ok := n.getWebhookConfig(ctx)
	if !ok {
		return
	}

	if n.isBatchEnabled(ctx) && len(deliveries) > 1 {
		n.deliverBatch(ctx, deliveries, webhookURL, secret)
	} else {
		for i := range deliveries {
			n.deliverOne(ctx, &deliveries[i], webhookURL, secret)
		}
	}
}

func (n *Notifier) isBatchEnabled(ctx context.Context) bool {
	var setting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_batch_enabled").First(&setting).Error; err != nil {
		return false
	}
	return setting.Value == "true"
}

func (n *Notifier) getWebhookConfig(ctx context.Context) (url, secret string, ok bool) {
	var urlSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_webhook_url").First(&urlSetting).Error; err != nil || urlSetting.Value == "" {
		return "", "", false
	}
	var secretSetting schemas.Setting
	if err := n.orm.WithContext(ctx).Where("key = ?", "nook_webhook_secret").First(&secretSetting).Error; err == nil {
		secret = secretSetting.Value
	}
	return urlSetting.Value, secret, true
}

func (n *Notifier) deliverOne(ctx context.Context, d *schemas.NookDelivery, webhookURL, secret string) {
	body := []byte(d.Payload)
	start := time.Now()
	code, respBody, err := n.send(ctx, webhookURL, secret, body)
	latency := int(time.Since(start).Milliseconds())

	d.Attempts++
	d.LatencyMs = &latency

	if err != nil {
		errMsg := err.Error()
		d.ErrorMessage = &errMsg
		n.scheduleRetry(d)
	} else if code >= 200 && code < 300 {
		d.Status = "delivered"
		d.ResponseCode = &code
		now := time.Now()
		d.DeliveredAt = &now
	} else if code >= 400 && code < 500 {
		d.Status = "failed"
		d.ResponseCode = &code
		trimmed := truncate(respBody, 1024)
		d.ResponseBody = &trimmed
	} else {
		d.ResponseCode = &code
		trimmed := truncate(respBody, 1024)
		d.ResponseBody = &trimmed
		n.scheduleRetry(d)
	}

	n.orm.WithContext(ctx).Save(d)
}

func (n *Notifier) scheduleRetry(d *schemas.NookDelivery) {
	retryIndex := d.Attempts - 1
	if retryIndex >= len(retryDelays) {
		d.Status = "failed"
		return
	}
	next := time.Now().Add(retryDelays[retryIndex])
	d.NextRetryAt = &next
	d.Status = "pending"
}

func (n *Notifier) deliverBatch(ctx context.Context, deliveries []schemas.NookDelivery, webhookURL, secret string) {
	events := make([]json.RawMessage, len(deliveries))
	for i, d := range deliveries {
		events[i] = json.RawMessage(d.Payload)
	}

	batchPayload, _ := json.Marshal(map[string]any{
		"batch":  true,
		"events": events,
	})

	start := time.Now()
	code, respBody, err := n.send(ctx, webhookURL, secret, batchPayload)
	latency := int(time.Since(start).Milliseconds())
	now := time.Now()

	for i := range deliveries {
		d := &deliveries[i]
		d.Attempts++
		d.LatencyMs = &latency

		if err != nil {
			errMsg := err.Error()
			d.ErrorMessage = &errMsg
			n.scheduleRetry(d)
		} else if code >= 200 && code < 300 {
			d.Status = "delivered"
			d.ResponseCode = &code
			d.DeliveredAt = &now
		} else if code >= 400 && code < 500 {
			d.Status = "failed"
			d.ResponseCode = &code
			trimmed := truncate(respBody, 1024)
			d.ResponseBody = &trimmed
		} else {
			d.ResponseCode = &code
			trimmed := truncate(respBody, 1024)
			d.ResponseBody = &trimmed
			n.scheduleRetry(d)
		}

		n.orm.WithContext(ctx).Save(d)
	}
}

func (n *Notifier) send(ctx context.Context, webhookURL, secret string, body []byte) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Nuage-Signature-256", sig)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return resp.StatusCode, string(respBytes), nil
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func (n *Notifier) ListDeliveries(ctx context.Context, limit, offset int) ([]schemas.NookDelivery, int64, error) {
	var total int64
	if err := n.orm.WithContext(ctx).Model(&schemas.NookDelivery{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var deliveries []schemas.NookDelivery
	if err := n.orm.WithContext(ctx).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&deliveries).Error; err != nil {
		return nil, 0, err
	}

	return deliveries, total, nil
}
