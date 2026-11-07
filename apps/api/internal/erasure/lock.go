package erasure

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/porte/directory"
)

// lockID namespaces the advisory lock that serialises deletion handling across
// processes. A start-first deploy runs the old and the new process side by
// side, and both would otherwise poll and receive pushes at once.
const lockID = 4920

type release func()

func acquire(ctx context.Context, db *sql.DB, wait bool) (release, bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	if wait {
		_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
		if err != nil {
			_ = conn.Close()
			return nil, false, err
		}
		return unlocker(conn), true, nil
	}
	var held bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&held); err != nil || !held {
		_ = conn.Close()
		return nil, false, err
	}
	return unlocker(conn), true, nil
}

func unlocker(conn *sql.Conn) release {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockID); err != nil {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
		_ = conn.Close()
	}
}

// Run polls at once, then every interval, until ctx ends. Each poll holds a
// Postgres advisory lock, so only one process polls at a time; a process that
// finds the lock taken skips that tick.
func (d *Deletions) Run(ctx context.Context) {
	ticker := time.NewTicker(directory.DefaultInterval)
	defer ticker.Stop()
	for {
		d.pollLocked(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *Deletions) pollLocked(ctx context.Context) {
	db, err := d.orm.DB()
	if err != nil {
		slog.Warn("erasure: no database handle for the poll lock", slog.Any("error", err))
		return
	}
	unlock, held, err := acquire(ctx, db, false)
	if err != nil || !held {
		if err != nil {
			slog.Warn("erasure: could not take the poll lock", slog.Any("error", err))
		}
		return
	}
	defer unlock()
	if err := d.receiver.Poll(ctx); err != nil {
		slog.Warn("directory: deletion poll failed, will retry", slog.Any("error", err))
	}
}

// Handler verifies and applies Registre's signed user.deleted push under the
// same advisory lock as the poll, so a push cannot overlap another process's
// poll or push. When the lock cannot be taken it answers 503 and Registre
// retries.
func (d *Deletions) Handler() http.Handler {
	inner := d.receiver.Handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		db, err := d.orm.DB()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		unlock, _, err := acquire(r.Context(), db, true)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		defer unlock()
		inner.ServeHTTP(w, r)
	})
}
