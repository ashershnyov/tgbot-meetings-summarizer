package storage

import "github.com/ashershnyov/tgbot-meetings-summarizer/internal/db"

// DB defines a database storage.
type DB struct {
	db.DB
}

// NewDB returns a new database storage.
func NewDB(db db.DB) DB {
	return DB{db}
}
