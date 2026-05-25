package activity

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Logger struct {
	orm *gorm.DB
}

func NewLogger(orm *gorm.DB) *Logger {
	return &Logger{orm: orm}
}

type Entry struct {
	UserID       int64
	EventType    string
	ResourceType string
	ResourceID   int64
	ResourceName string
	Metadata     map[string]any
}

func (l *Logger) Log(ctx context.Context, entry Entry) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("activity: panic in logger", slog.Any("panic", r))
			}
		}()
		l.insert(entry)
	}()
}

func (l *Logger) insert(entry Entry) {
	var metadataStr string
	if entry.Metadata != nil {
		raw, err := json.Marshal(entry.Metadata)
		if err == nil {
			metadataStr = string(raw)
		}
	}

	record := schemas.ActivityLog{
		UserID:       entry.UserID,
		EventType:    entry.EventType,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		ResourceName: entry.ResourceName,
		Metadata:     metadataStr,
	}

	if err := l.orm.Create(&record).Error; err != nil {
		slog.Warn("activity: failed to log entry",
			slog.String("event", entry.EventType),
			slog.Any("error", err),
		)
	}
}
