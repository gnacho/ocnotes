package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const initialSchema = `CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT 'New note',
    content TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    favorite INTEGER NOT NULL DEFAULT 0,
    modified INTEGER NOT NULL,
    etag TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notes_category ON notes(category);
CREATE INDEX IF NOT EXISTS idx_notes_favorite ON notes(favorite);
CREATE INDEX IF NOT EXISTS idx_notes_modified ON notes(modified);
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO settings (key, value) VALUES ('notesPath', 'Notes');
INSERT OR IGNORE INTO settings (key, value) VALUES ('fileSuffix', '.md');`

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

type Store struct {
	db *sql.DB
}

type Settings map[string]string

func Open(dataDir string) (*Store, error) {
	path := filepath.Join(dataDir, "notes.db")
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	db.SetMaxOpenConns(1)
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec("PRAGMA busy_timeout=5000")

	for _, stmt := range splitStatements(initialSchema) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("exec schema: %w", err)
		}
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

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

func (s *Store) GetNote(id int64) (*Note, error) {
	row := s.db.QueryRow(
		`SELECT id, etag, modified, title, category, content, favorite FROM notes WHERE id=?`, id,
	)
	return scanNote(row)
}

func (s *Store) ListNotes(category string, exclude []string, limit int, pruneBefore int64) ([]Note, error) {
	query := `SELECT id, etag, modified, title, category, content, favorite FROM notes`
	var args []interface{}
	where := ""

	if category != "" {
		where += " AND category=?"
		args = append(args, category)
	}

	if pruneBefore > 0 {
		if where == "" {
			where += " WHERE"
		} else {
			where += " AND"
		}
		where += " modified<?"
		args = append(args, pruneBefore)
	}

	if where != "" {
		query += where
	}

	query += ` ORDER BY favorite DESC, modified DESC`
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

func (s *Store) UpdateNote(id int64, title *string, content *string, category *string, favorite *bool, modified int64) (*Note, error) {
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

func (s *Store) DeleteNote(id int64) error {
	_, err := s.db.Exec(`DELETE FROM notes WHERE id=?`, id)
	return err
}

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

func (s *Store) GetSettings() (Settings, error) {
	settings := make(Settings)
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

func (s *Store) UpdateSettings(newSettings Settings) (Settings, error) {
	current, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	for key, value := range newSettings {
		if value == "" {
			switch key {
			case "notesPath":
				value = "Notes"
			case "fileSuffix":
				value = ".md"
			default:
				value = ""
			}
		}
		current[key] = value
		_, err := s.db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?,?)`, key, value)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

func splitStatements(sqlText string) []string {
	stmts := strings.Split(sqlText, ";")
	result := make([]string, 0, len(stmts))
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
