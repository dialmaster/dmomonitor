-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN admin tinyint(1) NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN admin;
-- +goose StatementEnd
