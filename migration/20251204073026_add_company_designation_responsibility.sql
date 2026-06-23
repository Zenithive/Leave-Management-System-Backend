-- +goose Up

-- Designation Table
CREATE TABLE IF NOT EXISTS Tbl_Designation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    designation_name VARCHAR(100) NOT NULL,
    description TEXT
);

-- Add designation_id column to Tbl_Employee
ALTER TABLE Tbl_Employee ADD COLUMN designation_id UUID;

-- Add foreign key constraint with ON DELETE SET NULL
ALTER TABLE Tbl_Employee 
ADD CONSTRAINT fk_employee_designation 
FOREIGN KEY (designation_id) 
REFERENCES Tbl_Designation(id) 
ON DELETE SET NULL;

-- +goose Down

-- Remove foreign key constraint
ALTER TABLE Tbl_Employee DROP CONSTRAINT IF EXISTS fk_employee_designation;

-- Remove designation_id column
ALTER TABLE Tbl_Employee DROP COLUMN IF EXISTS designation_id;

DROP TABLE IF EXISTS Tbl_Designation;
