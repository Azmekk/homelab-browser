package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/Azmekk/homelabbrowser/db"
	_ "modernc.org/sqlite"
)

//go:embed db/schema.sql
var schemaSQL string

type Store struct {
	DB      *sql.DB
	Queries *db.Queries
}

func openStore(dataDir string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		filepath.ToSlash(filepath.Join(dataDir, "app.db")))

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := sqlDB.ExecContext(context.Background(), schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("run schema: %w", err)
	}

	q := db.New(sqlDB)

	if _, err := q.GetSetting(context.Background(), settingPageTitle); err != nil {
		if err == sql.ErrNoRows {
			if err := q.UpsertSetting(context.Background(), db.UpsertSettingParams{
				Key:   settingPageTitle,
				Value: defaultPageTitle,
			}); err != nil {
				sqlDB.Close()
				return nil, fmt.Errorf("seed settings: %w", err)
			}
		} else {
			sqlDB.Close()
			return nil, fmt.Errorf("read page title: %w", err)
		}
	}

	return &Store{DB: sqlDB, Queries: q}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) PageTitle(ctx context.Context) string {
	v, err := s.Queries.GetSetting(ctx, settingPageTitle)
	if err != nil {
		return defaultPageTitle
	}
	return v
}
