-- +goose Up
-- +goose StatementBegin
CREATE TABLE meetings.user (
    id BIGINT PRIMARY KEY
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE meetings.user;
-- +goose StatementEnd
