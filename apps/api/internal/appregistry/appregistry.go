// Package appregistry holds the suite apps Nuage stores data for. An app is
// known by its Registre OIDC client id, configured by the operator in
// NUAGE_APPS, and proven on each request by the act.sub claim of the
// delegated JWT Registre minted for it. A label chosen when minting an API
// token never makes a caller an app.
package appregistry

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaxApps bounds how many apps one configuration may register.
const MaxApps = 32

var (
	keyPattern      = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)
	clientIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
)

// App is one registered app: a stable key used in URLs and storage, a name
// shown to users and the Registre client id its delegated tokens carry in
// act.sub.
type App struct {
	Key      string
	Name     string
	ClientID string
}

// Parse reads NUAGE_APPS: entries separated by ";", each
// "key:client_id[:Display name]". An empty value registers nothing. Any
// malformed entry, a duplicate key or client id, or a client id equal to
// ownClientID fails the whole value, so a typo stops the boot instead of
// silently leaving an app unable to act.
func Parse(raw, ownClientID string) ([]App, error) {
	var apps []App
	seenKey, seenClient := map[string]bool{}, map[string]bool{}
	for entry := range strings.SplitSeq(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		app, err := parseEntry(entry)
		if err != nil {
			return nil, err
		}
		switch {
		case seenKey[app.Key]:
			return nil, fmt.Errorf("NUAGE_APPS: app %q is declared twice", app.Key)
		case seenClient[app.ClientID]:
			return nil, fmt.Errorf("NUAGE_APPS: client id of %q is used by another app", app.Key)
		case ownClientID != "" && app.ClientID == ownClientID:
			return nil, fmt.Errorf("NUAGE_APPS: app %q cannot use Nuage's own client id", app.Key)
		}
		seenKey[app.Key], seenClient[app.ClientID] = true, true
		apps = append(apps, app)
	}
	if len(apps) > MaxApps {
		return nil, fmt.Errorf("NUAGE_APPS: at most %d apps", MaxApps)
	}
	return apps, nil
}

// ParseActors reads NUAGE_FULL_ACCESS_ACTORS: Registre client ids separated
// by "," or ";". A malformed id, or one that is also a registered app, fails
// the value: an app is confined to its folder and cannot also have the drive.
func ParseActors(raw string, apps []App) ([]string, error) {
	registered := map[string]bool{}
	for _, app := range apps {
		registered[app.ClientID] = true
	}
	var actors []string
	for _, actor := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' }) {
		actor = strings.TrimSpace(actor)
		switch {
		case actor == "":
			continue
		case !clientIDPattern.MatchString(actor):
			return nil, fmt.Errorf("NUAGE_FULL_ACCESS_ACTORS: %q must match %s", actor, clientIDPattern)
		case registered[actor]:
			return nil, fmt.Errorf("NUAGE_FULL_ACCESS_ACTORS: %q is a registered app", actor)
		}
		actors = append(actors, actor)
	}
	return actors, nil
}

// parseEntry reads one "key:client_id[:Display name]" entry.
func parseEntry(entry string) (App, error) {
	parts := strings.SplitN(entry, ":", 3)
	if len(parts) < 2 {
		return App{}, fmt.Errorf("NUAGE_APPS: %q is not key:client_id[:name]", entry)
	}
	app := App{Key: strings.TrimSpace(parts[0]), ClientID: strings.TrimSpace(parts[1])}
	if len(parts) == 3 {
		app.Name = strings.TrimSpace(parts[2])
	}
	if app.Name == "" {
		app.Name = app.Key
	}
	if !keyPattern.MatchString(app.Key) {
		return App{}, fmt.Errorf("NUAGE_APPS: key %q must match %s", app.Key, keyPattern)
	}
	if !clientIDPattern.MatchString(app.ClientID) {
		return App{}, fmt.Errorf("NUAGE_APPS: client id of %q must match %s", app.Key, clientIDPattern)
	}
	if len(app.Name) > 64 || strings.IndexFunc(app.Name, unicode.IsControl) >= 0 {
		return App{}, fmt.Errorf("NUAGE_APPS: name of %q must be at most 64 printable characters", app.Key)
	}
	return app, nil
}

// Sync makes the registered_apps table match apps: each one is upserted by
// key and marked active, and every row no longer configured is marked
// inactive. Rows are never deleted, because folders reference them.
func Sync(ctx context.Context, db *gorm.DB, apps []App) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		keys := []string{""}
		for _, app := range apps {
			if err := upsert(tx, app); err != nil {
				return err
			}
			keys = append(keys, app.Key)
		}
		return tx.Model(&schemas.RegisteredApp{}).Where("key NOT IN ?", keys).Update("active", false).Error
	})
}

// upsert registers app under its key. A row of another key still holding the
// same client id is retired first, so renaming an app's key keeps the client
// id unique.
func upsert(tx *gorm.DB, app App) error {
	if err := tx.Model(&schemas.RegisteredApp{}).Where("client_id = ? AND key <> ?", app.ClientID, app.Key).
		Updates(map[string]any{"client_id": gorm.Expr("'retired:' || id"), "active": false}).Error; err != nil {
		return err
	}
	row := schemas.RegisteredApp{Key: app.Key, Name: app.Name, ClientID: app.ClientID, Active: true}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "client_id", "active", "updated_at"}),
	}).Create(&row).Error
}

// ForClient returns the active app registered under clientID, or nil when no
// active app carries it.
func ForClient(ctx context.Context, db *gorm.DB, clientID string) (*schemas.RegisteredApp, error) {
	if clientID == "" {
		return nil, nil
	}
	var rows []schemas.RegisteredApp
	if err := db.WithContext(ctx).Where("client_id = ? AND active", clientID).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

type contextKey struct{}

// With returns ctx carrying the app the request acts as.
func With(ctx context.Context, app *schemas.RegisteredApp) context.Context {
	return context.WithValue(ctx, contextKey{}, app)
}

// From returns the app the request acts as, or nil for any other caller.
func From(ctx context.Context) *schemas.RegisteredApp {
	app, _ := ctx.Value(contextKey{}).(*schemas.RegisteredApp)
	return app
}
