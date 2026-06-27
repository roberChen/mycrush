-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN inherited_message_count INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN inherited_message_count;
-- +goose StatementEnd
