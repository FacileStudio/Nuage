package search

import (
	"context"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

type searchRow struct {
	ID        int64   `gorm:"column:id"`
	FacileID  string  `gorm:"column:facile_id"`
	Name      string  `gorm:"column:name"`
	Type      string  `gorm:"column:type"`
	MimeType  *string `gorm:"column:mime_type"`
	Size      int64   `gorm:"column:size"`
	FolderID  *int64  `gorm:"column:folder_id"`
	ParentID  *int64  `gorm:"column:parent_id"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (s *Service) Search(ctx context.Context, query string, filterType string, folderID *int64, limit int) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.Invalid("search query is required")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	pattern := "%" + strings.ToLower(query) + "%"

	var parts []string
	var args []any

	if filterType == "" || filterType == "file" {
		filePart := `SELECT id, facile_id, name, 'file' AS type, mime_type, size, folder_id, NULL::bigint AS parent_id, updated_at FROM files WHERE deleted_at IS NULL AND lower(name) LIKE ?`
		fileArgs := []any{pattern}

		if folderID != nil {
			filePart += ` AND folder_id = ?`
			fileArgs = append(fileArgs, *folderID)
		}

		parts = append(parts, filePart)
		args = append(args, fileArgs...)
	}

	if filterType == "" || filterType == "folder" {
		folderPart := `SELECT id, facile_id, name, 'folder' AS type, NULL::text AS mime_type, 0 AS size, NULL::bigint AS folder_id, parent_id, updated_at FROM folders WHERE deleted_at IS NULL AND lower(name) LIKE ?`
		folderArgs := []any{pattern}

		if folderID != nil {
			folderPart += ` AND parent_id = ?`
			folderArgs = append(folderArgs, *folderID)
		}

		parts = append(parts, folderPart)
		args = append(args, folderArgs...)
	}

	if len(parts) == 0 {
		return nil, errors.Invalid("type must be 'file', 'folder', or omitted")
	}

	sql := strings.Join(parts, " UNION ALL ")
	sql += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)

	var rows []searchRow
	if err := s.orm.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, errors.Internal("search failed", err)
	}

	folderIDs := map[int64]bool{}
	for _, r := range rows {
		if r.FolderID != nil {
			folderIDs[*r.FolderID] = true
		}
		if r.ParentID != nil {
			folderIDs[*r.ParentID] = true
		}
	}

	pathMap := s.buildPathMap(ctx, folderIDs)

	results := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		var parentPath string
		if r.Type == "file" && r.FolderID != nil {
			parentPath = pathMap[*r.FolderID]
		} else if r.Type == "folder" && r.ParentID != nil {
			parentPath = pathMap[*r.ParentID]
		}

		fullPath := "/" + r.Name
		if parentPath != "" {
			fullPath = parentPath + "/" + r.Name
		}

		mimeType := ""
		if r.MimeType != nil {
			mimeType = *r.MimeType
		}

		results = append(results, SearchResult{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			Type:      r.Type,
			Path:      fullPath,
			MimeType:  mimeType,
			Size:      r.Size,
			FolderID:  r.FolderID,
			ParentID:  r.ParentID,
			UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return &SearchResponse{Results: results, Total: len(results)}, nil
}

func (s *Service) buildPathMap(ctx context.Context, ids map[int64]bool) map[int64]string {
	if len(ids) == 0 {
		return map[int64]string{}
	}

	type folderRow struct {
		ID       int64  `gorm:"column:id"`
		Name     string `gorm:"column:name"`
		ParentID *int64 `gorm:"column:parent_id"`
	}

	var folders []folderRow
	s.orm.WithContext(ctx).Raw("SELECT id, name, parent_id FROM folders WHERE deleted_at IS NULL").Scan(&folders)

	lookup := map[int64]folderRow{}
	for _, f := range folders {
		lookup[f.ID] = f
	}

	cache := map[int64]string{}

	var resolve func(id int64) string
	resolve = func(id int64) string {
		if cached, ok := cache[id]; ok {
			return cached
		}
		f, ok := lookup[id]
		if !ok {
			return ""
		}
		if f.ParentID == nil {
			cache[id] = "/" + f.Name
		} else {
			parent := resolve(*f.ParentID)
			cache[id] = parent + "/" + f.Name
		}
		return cache[id]
	}

	result := map[int64]string{}
	for id := range ids {
		result[id] = resolve(id)
	}
	return result
}
