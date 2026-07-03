-- +goose Up
ALTER TABLE Tbl_Leave_type ADD COLUMN IF NOT EXISTS intern_entitlement INT;

-- +goose Down
ALTER TABLE Tbl_Leave_type DROP COLUMN IF EXISTS intern_entitlement;
