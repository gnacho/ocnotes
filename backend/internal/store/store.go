package store

import (
	_ "modernc.org/sqlite"
)

func scanNote(row dbRowScanner) (*Note, error) {
	var n Note
	var favInt int
	err := row.Scan(&n.ID, &n.Etag, &n.Modified, &n.Title, &n.Category, &n.Content, &favInt)
	n.Favorite = favInt == 1
	return &n, err
}

type dbRowScanner interface {
	Scan(dest ...interface{}) error
}
