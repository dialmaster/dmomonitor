-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN telegram_user_id VARCHAR(16) NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN telegram_user_id;
-- +goose StatementEnd
