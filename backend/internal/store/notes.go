// Package store handles SQLite database access for ocnotes.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// Note represents a Nextcloud-compatible note.
type Note struct {
	ID       int64  `json:"id"`
	Etag     string `json:"etag"`
	Readonly bool   `json:"readonly,omitempty"`
	Modified int64  `json:"modified"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Content  string `json:"content"`
	Favorite bool   `json:"favorite"`
}

// Store holds the open database and serves note operations.
type Store struct {
	db *sql.DB
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
		return nil, fmt.Errorf("max conns: %w", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec("PRAGMA busy_timeout=5000")

	migrations, err := listMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	for _, m := range migrations {
		sqlText, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", m, err)
		}
		for _, stmt := range splitStatements(string(sqlText)) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return nil, fmt.Errorf("exec %s: %w", m, err)
			}
		}
	}

	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// CreateNote inserts a new note and returns it with generated id + etag.
func (s *Store) CreateNote(title, content, category string, modified int64) (*Note, error) {
	tag := fmt.Sprintf("%x", modified)
	note := &Note{
		Title:    title,
		Content:  content,
		Category: category,
		Favorite: false,
		Modified: modified,
		Etag:     tag,
	}
	res, err := s.db.Exec(
		`INSERT INTO notes (title, content, category, favorite, modified, etag) VALUES (?,?,?,?,?,?)`,
		title, content, category, 0, modified, tag,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	note.ID = id
	return note, nil
}

// GetNote returns a single note by ID or sql.ErrNoRows.
func (s *Store) GetNote(id int64) (*Note, error) {
	row := s.db.QueryRow(
		`SELECT id, etag, modified, title, category, content, favorite FROM notes WHERE id=?`, id,
	)
	return scanNote(row)
}

// ListNotes returns up to limit notes ordered by favorite DESC, modified DESC.
// If category is non-empty, only notes in that category are returned.
func (s *Store) ListNotes(category string, exclude []string, limit int) ([]Note, error) {
	query := `SELECT id, etag, modified, title, category, content, favorite FROM notes`
	var args []interface{}
	where := ""

	if category != "" {
		where += " WHERE category=?"
		args = append(args, category)
	}

	query += where + ` ORDER BY favorite DESC, modified DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	return notes, rows.Err()
}

// UpdateNote updates a note's fields by ID. Returns the updated note or
// sql.ErrNoRows if not found.
func (s *Store) UpdateNote(id int64, title, content, category *string, favorite *bool, modified int64) (*Note, error) {
	// Read existing first for etag generation.
	existing, err := s.GetNote(id)
	if err != nil {
		return nil, err
	}

	newEtag := fmt.Sprintf("%x", modified)

	if title != nil && *title != existing.Title {
		existing.Title = *title
	}
	if content != nil {
		existing.Content = *content
	}
	if category != nil {
		existing.Category = *category
	}
	if favorite != nil {
		favInt := 0
		if *favorite {
			favInt = 1
		}
		s.db.Exec(`UPDATE notes SET favorite=? WHERE id=?`, favInt, id)
		existing.Favorite = *favorite
	}
	existing.Modified = modified
	existing.Etag = newEtag

	_, err = s.db.Exec(
		`UPDATE notes SET title=?, content=?, category=?, modified=?, etag=? WHERE id=?`,
		existing.Title, existing.Content, existing.Category, existing.Modified, existing.Etag, id,
	)
	if err != nil {
		return nil, err
	}
	return existing, nil
}

// DeleteNote deletes a note by ID. Returns sql.ErrNoRows if not found.
func (s *Store) DeleteNote(id int64) error {
	_, err := s.db.Exec(`DELETE FROM notes WHERE id=?`, id)
	return err
}

// SetFavorite toggles the favorite flag for a note.
func (s *Store) SetFavorite(id int64, favorite bool) (*Note, error) {
	favInt := 0
	if favorite {
		favInt = 1
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(
		`UPDATE notes SET favorite=?, modified=?, etag=? WHERE id=?`,
		favInt, now, fmt.Sprintf("%x", now), id,
	)
	if err != nil {
		return nil, err
	}
	return s.GetNote(id)
}

// SearchNotes searches title and content for the query string (case insensitive).
// Returns up to max results ordered by modified DESC.
func (s *Store) SearchNotes(query string, maxResults int) ([]Note, error) {
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, etag, modified, title, category, content, favorite FROM notes WHERE title LIKE ? OR content LIKE ? ORDER BY modified DESC LIMIT ?`,
		pattern, pattern, maxResults,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	return notes, rows.Err()
}
