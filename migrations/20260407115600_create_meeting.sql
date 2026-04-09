-- +goose Up
-- +goose StatementBegin
CREATE TABLE meetings.meeting (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES meetings.user(id) ON DELETE CASCADE,
    file_type TEXT NOT NULL,
    transcript TEXT NOT NULL,
    summary TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('russian', coalesce(transcript, ''))
    ) STORED
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE meetings.meeting;
-- +goose StatementEnd
