package quota

import (
	"context"
	"log/slog"
)

// freePlanBytes is the free plan's 5 GB ceiling, in bytes.
const freePlanBytes int64 = 5 << 30

// overRow is one person or space storing more than the free plan allows.
type overRow struct {
	ID       int64
	Name     string
	Ref      string
	Used     int64
	Override int64
}

// personalOverSQL lists the users whose personal live bytes exceed the free plan.
const personalOverSQL = `SELECT u.id, u.email AS name, COALESCE(i.subject, '') AS ref, t.used,
	COALESCE(q.storage_limit, 0) AS override
	FROM (SELECT f.uploaded_by AS user_id, SUM(f.size + COALESCE(
		(SELECT SUM(v.size) FROM file_versions v WHERE v.file_id = f.id), 0)) AS used
		FROM files f WHERE f.deleted_at IS NULL AND f.space_id IS NULL GROUP BY f.uploaded_by) t
	JOIN users u ON u.id = t.user_id
	LEFT JOIN porte_identities i ON i.user_id = u.id AND i.provider = ?
	LEFT JOIN user_quotas q ON q.user_id = u.id
	WHERE t.used > ? ORDER BY t.used DESC`

// spaceOverSQL lists the spaces whose live bytes exceed the free plan.
const spaceOverSQL = `SELECT s.id, s.name, COALESCE(s.registre_space_id, '') AS ref, t.used, 0 AS override
	FROM (SELECT f.space_id, SUM(f.size + COALESCE(
		(SELECT SUM(v.size) FROM file_versions v WHERE v.file_id = f.id), 0)) AS used
		FROM files f WHERE f.deleted_at IS NULL AND f.space_id IS NOT NULL GROUP BY f.space_id) t
	JOIN spaces s ON s.id = t.space_id
	WHERE t.used > ? ORDER BY t.used DESC`

// ReportOverFreePlan logs, once at boot, every person and space whose stored
// bytes exceed the free plan. Before Ardoise's first push the owner adds an
// override for each of them, so no one is cut off from new uploads the day
// plans start applying. A person with an admin override (override != 0) keeps
// that override and needs nothing.
func (s *Service) ReportOverFreePlan(ctx context.Context, logger *slog.Logger) {
	issuer := ""
	if s.plans != nil {
		issuer = s.plans.Issuer()
	}
	var people, spaces []overRow
	if err := s.orm.WithContext(ctx).Raw(personalOverSQL, issuer, freePlanBytes).Scan(&people).Error; err != nil {
		logger.Warn("forfait: could not list people over the free plan", slog.Any("error", err))
	}
	if err := s.orm.WithContext(ctx).Raw(spaceOverSQL, freePlanBytes).Scan(&spaces).Error; err != nil {
		logger.Warn("forfait: could not list spaces over the free plan", slog.Any("error", err))
	}
	for _, row := range people {
		logger.Warn("forfait: personal storage above the free plan", slog.Int64("user_id", row.ID),
			slog.String("email", row.Name), slog.String("subject", row.Ref),
			slog.Int64("used_bytes", row.Used), slog.Int64("admin_override_bytes", row.Override))
	}
	for _, row := range spaces {
		logger.Warn("forfait: space storage above the free plan", slog.Int64("space_id", row.ID),
			slog.String("name", row.Name), slog.String("registre_space_id", row.Ref), slog.Int64("used_bytes", row.Used))
	}
}
