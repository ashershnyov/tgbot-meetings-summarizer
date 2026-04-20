-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA meetings;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA meetings CASCADE;
-- +goose StatementEnd
