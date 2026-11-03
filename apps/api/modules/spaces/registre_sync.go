package spaces

import (
	"context"
	"log/slog"
	"time"

	"github.com/FacileStudio/porte/directory"
)

// retryPolicy is how long Run waits after a failed import: first, doubling each time up to
// max. Registre may deploy after Nuage and answer 404 for a while, so the ceiling keeps a
// long outage to one warning every few minutes.
type retryPolicy struct {
	first time.Duration
	max   time.Duration
}

const (
	retryFirst = 10 * time.Second
	retryMax   = 5 * time.Minute
)

// Run imports Nuage's spaces into Registre, retrying until it succeeds, and only then
// starts the syncer. The order matters: a sync before the import would remove every member
// Registre has not yet heard of. It returns when ctx ends.
func (r *Registre) Run(ctx context.Context, syncer *directory.Syncer) {
	delay := r.retry.first
	for {
		err := r.Import(ctx, syncer.Client)
		if err == nil {
			break
		}
		r.logger.Warn("directory: import into registre failed, retrying",
			slog.Any("error", err), slog.Duration("retry_in", delay))
		if !wait(ctx, delay) {
			return
		}
		delay = min(delay*2, r.retry.max)
	}
	syncer.Run(ctx)
}

// Import hands Registre every user-made space not yet linked to it, with its members.
// Registre records the import once per client, so calling it again after a restart is a
// no-op.
func (r *Registre) Import(ctx context.Context, client directory.Client) error {
	spaces, err := r.importSpaces(ctx)
	if err != nil {
		return err
	}
	_, err = client.Import(ctx, spaces)
	return err
}

// importRow is one membership of an unlinked space, with the member's subject under the
// issuer when they have one.
type importRow struct {
	SpaceID int64
	Name    string
	Email   string
	Subject string
	Role    string
}

// importSpaces reads the unlinked spaces and their members, oldest space first. A member is
// named by subject when they hold an identity under the issuer, and by email otherwise.
// Spaces another app provisioned (provisioned_by set) are that app's publishing targets,
// not team spaces, and never reach Registre.
func (r *Registre) importSpaces(ctx context.Context) ([]directory.ImportSpace, error) {
	var rows []importRow
	err := r.orm.WithContext(ctx).Table("spaces").
		Select("spaces.id AS space_id, spaces.name, users.email, COALESCE(i.subject, '') AS subject, space_members.role").
		Joins("JOIN space_members ON space_members.space_id = spaces.id").
		Joins("JOIN users ON users.id = space_members.user_id").
		Joins("LEFT JOIN porte_identities i ON i.user_id = users.id AND i.provider = ?", r.issuer).
		Where("spaces.registre_space_id IS NULL AND spaces.provisioned_by = ''").
		Order("spaces.created_at asc, spaces.id, space_members.joined_at asc, space_members.id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return groupImport(rows), nil
}

// groupImport folds the rows into one ImportSpace per local space, in row order.
func groupImport(rows []importRow) []directory.ImportSpace {
	spaces := []directory.ImportSpace{}
	index := map[int64]int{}
	for _, row := range rows {
		at, ok := index[row.SpaceID]
		if !ok {
			at = len(spaces)
			index[row.SpaceID] = at
			spaces = append(spaces, directory.ImportSpace{Name: row.Name})
		}
		member := directory.ImportMember{Subject: row.Subject, Role: localRole(row.Role)}
		if member.Subject == "" {
			member.Email = row.Email
		}
		spaces[at].Members = append(spaces[at].Members, member)
	}
	return spaces
}

// wait sleeps for d or until ctx ends, reporting whether it slept fully.
func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
