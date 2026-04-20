package storage

import (
	"context"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/model"
)

const qAddUser = `INSERT INTO meetings.user (id) VALUES ($1) ON CONFLICT DO NOTHING;`

// AddUser registers a new user. Would do nothing if user already exists.
func (db *DB) AddUser(ctx context.Context, userID int64) error {
	_, err := db.ExecContext(ctx, qAddUser, userID)
	return err
}

const qAddMeeting = `INSERT INTO meetings.meeting (user_id, file_type, transcript) VALUES ($1, $2, $3) RETURNING id;;`

// AddMeeting adds a new meeting.
func (db *DB) AddMeeting(ctx context.Context, m *model.Meeting) (string, error) {
	meeting := model.Meeting{}
	err := db.QueryOneContext(ctx, &meeting, qAddMeeting, m.UserID, m.FileType, m.Transcript)
	return meeting.ID, err
}

const qUpdateSummary = `UPDATE meetings.meeting SET summary = $1 WHERE id = $2;`

// UpdateSummary changes summary for an already existing meeting.
func (db *DB) UpdateSummary(ctx context.Context, id, summary string) error {
	_, err := db.ExecContext(ctx, qUpdateSummary, summary, id)
	return err
}
