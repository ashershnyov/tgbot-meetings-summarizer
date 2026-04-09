-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_meetings_search ON meetings.meeting USING GIN(search_vector);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_meetings_search;
-- +goose StatementEnd
