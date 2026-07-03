-- +goose Up
-- Add ending_date column to Tbl_Employee
ALTER TABLE Tbl_Employee ADD COLUMN ending_date DATE DEFAULT NULL;

-- +goose Down
-- Remove ending_date column
ALTER TABLE Tbl_Employee DROP COLUMN IF EXISTS ending_date;
