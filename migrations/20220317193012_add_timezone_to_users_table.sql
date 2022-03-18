-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN timezone VARCHAR(32) NOT NULL DEFAULT 'US/Eastern';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN timezone;
-- +goose StatementEnd
