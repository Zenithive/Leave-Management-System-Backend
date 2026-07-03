-- +goose Up
ALTER TABLE Tbl_Company_Settings ADD COLUMN IF NOT EXISTS company_name VARCHAR(255) DEFAULT 'DEMO';
ALTER TABLE Tbl_Company_Settings ADD COLUMN IF NOT EXISTS logo_path TEXT DEFAULT '';
ALTER TABLE Tbl_Company_Settings ADD COLUMN IF NOT EXISTS primary_color VARCHAR(7) DEFAULT '#2980b9';
ALTER TABLE Tbl_Company_Settings ADD COLUMN IF NOT EXISTS secondary_color VARCHAR(7) DEFAULT '#2ecc71';

-- +goose Down
ALTER TABLE Tbl_Company_Settings DROP COLUMN IF EXISTS company_name;
ALTER TABLE Tbl_Company_Settings DROP COLUMN IF EXISTS logo_path;
ALTER TABLE Tbl_Company_Settings DROP COLUMN IF EXISTS primary_color;
ALTER TABLE Tbl_Company_Settings DROP COLUMN IF EXISTS secondary_color;

