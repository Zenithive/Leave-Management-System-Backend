-- +goose Up
-- Fix absent_days column to support decimal values (e.g., 2.5 for half days)
ALTER TABLE Tbl_Payslip ALTER COLUMN absent_days TYPE NUMERIC;

-- +goose Down
-- Revert back to INT (this will truncate decimal values)
ALTER TABLE Tbl_Payslip ALTER COLUMN absent_days TYPE INT;