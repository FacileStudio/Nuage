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

// chargedBytesSQL sums every file's bytes and versions by who pays for them:
// the person for personal files and app-provisioned spaces charged to them,
// the space for every other space.
const chargedBytesSQL = `SELECT CASE WHEN f.space_id IS NULL THEN f.uploaded_by
		WHEN sp.registre_space_id IS NULL AND sp.charged_to IS NOT NULL THEN sp.charged_to END AS user_id,
	CASE WHEN f.space_id IS NOT NULL AND (sp.registre_space_id IS NOT NULL OR sp.charged_to IS NULL)
		THEN f.space_id END AS space_id,
	SUM(f.size + COALESCE((SELECT SUM(v.size) FROM file_versions v WHERE v.file_id = f.id), 0)) AS used
	FROM files f LEFT JOIN spaces sp ON sp.id = f.space_id GROUP BY 1, 2`

// personalOverSQL lists the people whose charged bytes exceed the free plan,
// with every subject they hold under the issuer, most recent first.
const personalOverSQL = `SELECT u.id, u.email AS name,
	COALESCE((SELECT string_agg(i.subject, ',' ORDER BY i.synced_at DESC NULLS LAST, i.created_at DESC NULLS LAST)
		FROM porte_identities i WHERE i.user_id = u.id AND i.provider = ?), '') AS ref,
	SUM(t.used) AS used,
	COALESCE((SELECT q.storage_limit FROM user_quotas q WHERE q.user_id = u.id), 0) AS override
	FROM (` + chargedBytesSQL + `) t
	JOIN users u ON u.id = t.user_id
	GROUP BY u.id, u.email HAVING SUM(t.used) > ? ORDER BY used DESC`

// spaceOverSQL lists the spaces paying for themselves whose bytes exceed the
// free plan.
const spaceOverSQL = `SELECT s.id, s.name, COALESCE(s.registre_space_id, '') AS ref, SUM(t.used) AS used, 0 AS override
	FROM (` + chargedBytesSQL + `) t
	JOIN spaces s ON s.id = t.space_id
	GROUP BY s.id, s.name, s.registre_space_id HAVING SUM(t.used) > ? ORDER BY used DESC`

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
			slog.String("email", row.Name), slog.String("subjects", row.Ref),
			slog.Int64("used_bytes", row.Used), slog.Int64("admin_override_bytes", row.Override))
	}
	for _, row := range spaces {
		logger.Warn("forfait: space storage above the free plan", slog.Int64("space_id", row.ID),
			slog.String("name", row.Name), slog.String("registre_space_id", row.Ref), slog.Int64("used_bytes", row.Used))
	}
}
