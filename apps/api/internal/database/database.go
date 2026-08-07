package database

import (
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// statementTimeout bounds any single query so a pathological plan cannot pin a
// backend connection indefinitely and exhaust the pool.
const statementTimeout = 30 * time.Second

// withStatementTimeout folds the timeout into the DSN's libpq options so every
// connection the pool opens carries it. Issuing SET once after Open would apply
// to whichever single pooled connection happened to serve it, and SET takes no
// bind parameter, so the parameterised form is a syntax error rather than a
// slow leak.
func withStatementTimeout(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	if query.Get("options") != "" {
		return databaseURL, nil
	}
	query.Set("options", fmt.Sprintf("-c statement_timeout=%d", statementTimeout.Milliseconds()))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func Open(databaseURL string) (*gorm.DB, error) {
	dsn, err := withStatementTimeout(databaseURL)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return db, nil
}
