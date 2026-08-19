// Package store handles SQLite database access for ocnotes.
//
// Schema is versioned via PRAGMA user_version. Migrations live in
// migrations/*.sql files named NNN_description.sql. The database is opened
// with MaxOpenConns(1) (single writer), WAL mode, and synchronous normal.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"strconv"

	_ "modernc.org/sqlite"
)

// Store holds the open database handle and the current schema version.
type Store struct {
	db  *sql.DB
	ctx *context
}

// Open opens or creates the Notes database at dataDir/notes.db. Creates tables
// if missing, runs all pending migrations, and sets PRAGMAs.
func Open(dataDir string) (*Store, error) {
	path := filepath.Join(dataDir, "notes.db")
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := db.SetMaxOpenConns(1); err != nil {
		return nil, fmt.Errorf("set max conns: %w", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec("PRAGMA busy_timeout=5000")

	// Apply migrations.
	version := 0
	var row interface{}
	if err := db.QueryRow("PRAGMA user_version").Scan(&row); err == nil {
		version = row.(int)
	}

	migrations, err := listMigrations(version)
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	for _, m := range migrations {
		if err := runMigration(db, m); err != nil {
			return nil, fmt.Errorf("migration %s: %w", m, err)
		}
		version++
		db.Exec(fmt.Sprintf("PRAGMA user_version=%d", version))
	}

	return &Store{db: db, ctx: newContext()}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Context returns the application context for shadow-user operations.
func (s *Store) Context() *context { return s.ctx }

// DB returns the underlying *sql.DB for advanced usage (migrations, health checks).
func (s *Store) DB() *sql.DB { return s.db }
