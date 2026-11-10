package erasure

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"

	"gorm.io/gorm"
)

// PersonalFiles removes, inside the erasure's transaction, the files and
// folders a person keeps outside any space, and returns the removal of their
// stored bytes for after the commit.
type PersonalFiles interface {
	ErasePersonal(ctx context.Context, tx *gorm.DB, userID int64) (func(context.Context), error)
}

// AppData removes, inside the erasure's transaction, every app folder a
// person holds, and returns the removal of their stored bytes for after the
// commit.
type AppData interface {
	EraseAll(ctx context.Context, tx *gorm.DB, userID int64) (func(context.Context), error)
}

// Deletions erases the people Registre erased. It polls Registre's deletion
// list with the same client credentials as the snapshot sync and accepts
// Registre's signed push on directory.DeletionsPath.
type Deletions struct {
	orm        *gorm.DB
	files      PersonalFiles
	apps       AppData
	storageDir string
	accounts   *kit.Store
	receiver   *directory.Deletions
}

// Config wires the receiver to Nuage: the database, Registre's directory client
// (the same credentials as the snapshot sync), porte's own rows eraser
// (pg.Store.EraseIdentity), the two stores that hold a person's files, and the
// directory the avatars live in.
type Config struct {
	ORM        *gorm.DB
	Client     directory.Client
	Erase      func(ctx context.Context, provider, subject string) error
	Files      PersonalFiles
	Apps       AppData
	StorageDir string
	Logger     *slog.Logger
}

// New builds the receiver over Nuage's tables. The account itself is erased
// through porte/kit, which resolves the identity with or without the issuer's
// trailing slash and runs eraseData and the users delete in one transaction
// under the lock every sign-in takes.
func New(ctx context.Context, cfg Config) (*Deletions, error) {
	sqlDB, err := cfg.ORM.DB()
	if err != nil {
		return nil, err
	}
	d := &Deletions{orm: cfg.ORM, files: cfg.Files, apps: cfg.Apps, storageDir: cfg.StorageDir}
	d.accounts, err = kit.NewStore(ctx, kit.Store{DB: sqlDB, Columns: auth.StoreColumns(), EraseData: d.eraseData, Logger: cfg.Logger})
	if err != nil {
		return nil, err
	}
	d.receiver = &directory.Deletions{
		Client:          cfg.Client,
		OnPersonDeleted: d.ErasePerson,
		Erase:           cfg.Erase,
		State:           NewState(cfg.ORM),
		Logger:          cfg.Logger,
	}
	return d, nil
}

type account struct {
	ID         int64
	Email      string
	UploadPath string
}

type cleanupKey struct{}

// ErasePerson deletes the account behind a Registre identity and what it owns,
// in one transaction: personal files and folders, app folders, shares, tokens,
// quota, then the account. Space content it authored stays, reassigned to the
// Deleted user placeholder, and a space it owned alone among its members gets
// its longest-standing member as owner. Stored bytes and the uploaded avatar
// are removed after the commit. A subject that is already gone is a no-op, so
// a replayed notice changes nothing.
func (d *Deletions) ErasePerson(ctx context.Context, provider, subject string) error {
	var cleanups []func(context.Context)
	if err := d.accounts.EraseAccount(context.WithValue(ctx, cleanupKey{}, &cleanups), provider, subject); err != nil {
		return err
	}
	for _, cleanup := range cleanups {
		cleanup(ctx)
	}
	return nil
}

func (d *Deletions) eraseData(ctx context.Context, sqlTx *sql.Tx, userID int64) error {
	tx := d.orm.WithContext(ctx)
	tx.Statement.ConnPool = sqlTx
	var found account
	if err := tx.Raw(`SELECT id, email, coalesce(avatar_upload_path, '') AS upload_path FROM users WHERE id = ?`,
		userID).Scan(&found).Error; err != nil {
		return err
	}
	removeFiles, err := d.files.ErasePersonal(ctx, tx, userID)
	if err != nil {
		return err
	}
	removeApps, err := d.apps.EraseAll(ctx, tx, userID)
	if err != nil {
		return err
	}
	if err := deleteAccount(tx, found); err != nil {
		return err
	}
	if cleanups, ok := ctx.Value(cleanupKey{}).(*[]func(context.Context)); ok {
		*cleanups = append(*cleanups, removeFiles, removeApps, func(context.Context) { removeUpload(d.storageDir, found.UploadPath) })
	}
	return nil
}

func removeUpload(storageDir, relative string) {
	if relative == "" {
		return
	}
	path := filepath.Join(storageDir, "avatars", filepath.Base(relative))
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("failed to remove an erased person's avatar", slog.Any("error", err))
	}
}
