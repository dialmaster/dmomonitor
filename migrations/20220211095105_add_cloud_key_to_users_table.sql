-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN cloud_key VARCHAR(64) not null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN cloud_key;
-- +goose StatementEnd
