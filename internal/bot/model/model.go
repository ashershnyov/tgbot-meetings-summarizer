package model

import "time"

// User describes one user.
type User struct {
	ID int64 `db:"id"`
}

// Meeting describes one meeting.
type Meeting struct {
	ID         string    `db:"id"`
	UserID     int64     `db:"user_id"`
	FileType   string    `db:"file_type"`
	Transcript string    `db:"transcript"`
	Summary    *string   `db:"summary"`
	CreatedAt  time.Time `db:"created_at"`
}
