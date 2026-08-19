package store

import (
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanNote(row rowScanner) (*Note, error) {
	var n Note
	var favInt int
	err := row.Scan(&n.ID, &n.Etag, &n.Modified, &n.Title, &n.Category, &n.Content, &favInt)
	n.Favorite = favInt == 1
	return &n, err
}

type prunedNote struct {
	ID int64 `json:"id"`
}

type dbRowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPrunedNote(row dbRowScanner) (*prunedNote, error) {
	var pn prunedNote
	err := row.Scan(&pn.ID)
	return &pn, err
}

func listMigrations() ([]string, error) {
	dir := filepath.Join("internal", "store", "migrations")
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var migrations []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			migrations = append(migrations, filepath.Join(dir, f.Name()))
		}
	}
	return migrations, nil
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
