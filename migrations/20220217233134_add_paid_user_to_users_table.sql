-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN paid tinyint(1) NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN paid;
-- +goose StatementEnd
