-- +goose Up
-- +goose StatementBegin

UPDATE Tbl_Role
SET type = 'MANAGER'
WHERE type = 'MANAGAR';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
