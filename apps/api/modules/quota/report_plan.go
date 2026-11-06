package quota

import (
	"context"
	"log/slog"
	"strings"

	"github.com/FacileStudio/porte/forfait"
)

// chargedPeopleSQL sums the bytes charged to each person.
const chargedPeopleSQL = `SELECT t.user_id AS id, SUM(t.used) AS used FROM (` + chargedBytesSQL + `) t
	WHERE t.user_id IS NOT NULL GROUP BY t.user_id`

// chargedSpacesSQL sums the bytes of each space linked to Registre.
const chargedSpacesSQL = `SELECT s.id, s.name, s.registre_space_id AS ref, SUM(t.used) AS used FROM (` + chargedBytesSQL + `) t
	JOIN spaces s ON s.id = t.space_id WHERE s.registre_space_id IS NOT NULL GROUP BY s.id, s.name, s.registre_space_id`

// identitiesSQL lists the users holding more than one identity under the
// issuer, with their subjects most recent first.
const identitiesSQL = `SELECT u.id, u.email AS name,
	string_agg(i.subject, ',' ORDER BY i.synced_at DESC NULLS LAST, i.created_at DESC NULLS LAST, i.subject) AS ref
	FROM porte_identities i JOIN users u ON u.id = i.user_id WHERE i.provider = ?
	GROUP BY u.id, u.email HAVING count(*) > 1 ORDER BY u.id`

// ReportIdentities logs every user holding several identities under the
// issuer, and the subject their plan is read under. Nothing is deleted: an
// operator removes a stale identity by hand once it is confirmed unused.
func (s *Service) ReportIdentities(ctx context.Context, logger *slog.Logger) {
	var rows []overRow
	if err := s.orm.WithContext(ctx).Raw(identitiesSQL, s.plans.Issuer()).Scan(&rows).Error; err != nil {
		logger.Warn("forfait: could not list users with several identities", slog.Any("error", err))
		return
	}
	for _, row := range rows {
		logger.Warn("forfait: user holds several identities under the issuer", slog.Int64("user_id", row.ID),
			slog.String("email", row.Name), slog.String("subjects", row.Ref),
			slog.String("plan_subject", s.pickSubject(strings.Split(row.Ref, ","))))
	}
}

// ReportOverPlan logs, once plans apply, every person and Registre-linked
// space storing more than the ceiling it now resolves to: the plan, or a
// person's admin override. Each one is refused new uploads until an Ardoise
// override or an upgrade covers it.
func (s *Service) ReportOverPlan(ctx context.Context, logger *slog.Logger) {
	if !s.plans.Enforcing() {
		return
	}
	s.reportPeopleOverPlan(ctx, logger)
	s.reportSpacesOverPlan(ctx, logger)
}

// reportPeopleOverPlan logs each person over their resolved ceiling.
func (s *Service) reportPeopleOverPlan(ctx context.Context, logger *slog.Logger) {
	var people []overRow
	if err := s.orm.WithContext(ctx).Raw(chargedPeopleSQL).Scan(&people).Error; err != nil {
		logger.Warn("forfait: could not list people's usage", slog.Any("error", err))
		return
	}
	for _, row := range people {
		limit, err := s.personCeiling(ctx, row.ID)
		if err == nil && limit >= 0 && row.Used > limit {
			logger.Warn("forfait: person over their plan", slog.Int64("user_id", row.ID),
				slog.Int64("used_bytes", row.Used), slog.Int64("limit_bytes", limit))
		}
	}
}

// reportSpacesOverPlan logs each Registre-linked space over its plan.
func (s *Service) reportSpacesOverPlan(ctx context.Context, logger *slog.Logger) {
	var spaces []overRow
	if err := s.orm.WithContext(ctx).Raw(chargedSpacesSQL).Scan(&spaces).Error; err != nil {
		logger.Warn("forfait: could not list spaces' usage", slog.Any("error", err))
		return
	}
	for _, row := range spaces {
		limit := s.plans.LimitBytes(forfait.Scope{SpaceID: row.Ref})
		if limit >= 0 && row.Used > limit {
			logger.Warn("forfait: space over its plan", slog.Int64("space_id", row.ID), slog.String("name", row.Name),
				slog.String("registre_space_id", row.Ref), slog.Int64("used_bytes", row.Used), slog.Int64("limit_bytes", limit))
		}
	}
}

// personCeiling is the ceiling a person's uploads are held to, in bytes: their
// admin override when there is one, else their plan; -1 is unlimited.
func (s *Service) personCeiling(ctx context.Context, userID int64) (int64, error) {
	db := s.orm.WithContext(ctx)
	override, err := overrideOf(db, userID)
	if err != nil || override != 0 {
		return override, err
	}
	person, err := s.personTarget(db, userID)
	if err != nil {
		return 0, err
	}
	return s.plans.LimitBytes(person.scope), nil
}
