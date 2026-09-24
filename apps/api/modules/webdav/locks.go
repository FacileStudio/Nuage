package webdav

import (
	"fmt"
	"sync"

	"golang.org/x/net/webdav"
)

// lockRegistry hands each mount its own lock system. The webdav library keys
// locks on the stripped path, so one shared LockSystem would let a lock taken on
// /webdav/a.txt apply to /webdav/spaces/3/a.txt, and let one user's token be
// replayed by another.
type lockRegistry struct {
	mu    sync.Mutex
	locks map[string]webdav.LockSystem
}

// newLockRegistry builds an empty lock registry.
func newLockRegistry() *lockRegistry {
	return &lockRegistry{locks: make(map[string]webdav.LockSystem)}
}

// forKey returns the lock system for a mount key, creating it on first use.
func (l *lockRegistry) forKey(key string) webdav.LockSystem {
	l.mu.Lock()
	defer l.mu.Unlock()
	ls, ok := l.locks[key]
	if !ok {
		ls = webdav.NewMemLS()
		l.locks[key] = ls
	}
	return ls
}

// lockKey namespaces a lock system per mount so tokens never cross mounts. A
// space mount is keyed by the space alone, not by the caller: every member has
// to share one lock table, or a lock taken by one member would not stop another
// from overwriting the file it guards.
func lockKey(userID int64, m mountPath) string {
	switch m.kind {
	case mountSpace:
		return fmt.Sprintf("s%d", m.spaceID)
	case mountIndex:
		return fmt.Sprintf("u%d/index", userID)
	default:
		return fmt.Sprintf("u%d/root", userID)
	}
}
