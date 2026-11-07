package erasure

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/FacileStudio/porte/directory"

	"gorm.io/gorm"
)

// PersonalFiles removes the files and folders a person keeps outside any space.
type PersonalFiles interface {
	ErasePersonal(ctx context.Context, userID int64) error
}

// AppData removes every app folder a person holds.
type AppData interface {
	DeleteAll(ctx context.Context, userID int64) error
}

// Deletions erases the people Registre erased. It polls Registre's deletion
// list with the same client credentials as the snapshot sync and accepts
// Registre's signed push on directory.DeletionsPath.
type Deletions struct {
	orm        *gorm.DB
	files      PersonalFiles
	apps       AppData
	storageDir string
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

// New builds the receiver over Nuage's tables.
func New(cfg Config) *Deletions {
	d := &Deletions{orm: cfg.ORM, files: cfg.Files, apps: cfg.Apps, storageDir: cfg.StorageDir}
	d.receiver = &directory.Deletions{
		Client:          cfg.Client,
		OnPersonDeleted: d.ErasePerson,
		Erase:           cfg.Erase,
		State:           NewState(cfg.ORM),
		Logger:          cfg.Logger,
	}
	return d
}

type account struct {
	ID         int64
	Email      string
	UploadPath string
}

// ErasePerson deletes the account behind a Registre identity and what it owns:
// personal files and folders, app folders, tokens and quota. Space content it
// authored stays, reassigned to the Deleted user placeholder. A subject that is
// already gone is a no-op, so a replayed notice changes nothing.
func (d *Deletions) ErasePerson(ctx context.Context, provider, subject string) error {
	found, err := d.find(ctx, provider, subject)
	if err != nil || found.ID == 0 {
		return err
	}
	if err := d.files.ErasePersonal(ctx, found.ID); err != nil {
		return err
	}
	if err := d.apps.DeleteAll(ctx, found.ID); err != nil {
		return err
	}
	if err := d.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteAccount(tx, found)
	}); err != nil {
		return err
	}
	removeUpload(d.storageDir, found.UploadPath)
	return nil
}

func (d *Deletions) find(ctx context.Context, provider, subject string) (account, error) {
	var found account
	err := d.orm.WithContext(ctx).Raw(
		`SELECT u.id, u.email, coalesce(u.avatar_upload_path, '') AS upload_path FROM users u
		 JOIN porte_identities i ON i.user_id = u.id WHERE i.provider = ? AND i.subject = ?`,
		provider, subject).Scan(&found).Error
	return found, err
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
