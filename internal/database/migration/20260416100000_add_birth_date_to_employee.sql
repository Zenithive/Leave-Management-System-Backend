-- +goose Up
ALTER TABLE Tbl_Employee ADD COLUMN birth_date DATE DEFAULT NULL;

-- +goose Down
ALTER TABLE Tbl_Employee DROP COLUMN IF EXISTS birth_date;
