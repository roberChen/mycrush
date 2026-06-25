-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN badge TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN badge;
-- +goose StatementEnd
