-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_meeting_user_date ON meetings.meeting(user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_meeting_user_date;
-- +goose StatementEnd
