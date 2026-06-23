-- +goose Up
-- +goose StatementBegin
ALTER TABLE Tbl_Leave 
ALTER COLUMN reason SET DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
