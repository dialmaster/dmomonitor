-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN last_active int(11) NOT NULL DEFAULT(0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN last_active;
-- +goose StatementEnd
