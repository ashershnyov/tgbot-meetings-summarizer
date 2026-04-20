package storage

import (
	"context"

	"github.com/ashershnyov/tgbot-meetings-summarizer/internal/bot/model"
)

const qGetMeeting = `SELECT id, file_type, transcript, summary, created_at FROM meetings.meeting WHERE id = $1 AND user_id = $2;`

// GetMeeting gets the meeting by id for the given user.
func (db *DB) GetMeeting(ctx context.Context, userID int64, id string) (model.Meeting, error) {
	meeting := model.Meeting{}
	err := db.QueryOneContext(ctx, &meeting, qGetMeeting, id, userID)
	return meeting, err
}

const qListMeetings = `SELECT id, created_at, summary FROM meetings.meeting WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20;`

// ListMeetings returns 20 last meetings added by user.
func (db *DB) ListMeetings(ctx context.Context, userID int64) ([]model.Meeting, error) {
	meetings := []model.Meeting{}
	err := db.QueryManyContext(ctx, &meetings, qListMeetings, userID)
	if err != nil {
		return nil, err
	}
	return meetings, nil
}

const qFindMeetings = `
	SELECT id, created_at, summary FROM meetings.meeting 
	WHERE user_id = $1 AND search_vector @@ plainto_tsquery('russian', $2) 
	ORDER BY created_at DESC LIMIT 20;
`

// FindMeetings returns meetings by query for the given user.
func (db *DB) FindMeetings(ctx context.Context, userID int64, query string) ([]model.Meeting, error) {
	meetings := []model.Meeting{}
	err := db.QueryManyContext(ctx, &meetings, qFindMeetings, userID, query)
	if err != nil {
		return nil, err
	}
	return meetings, nil
}
