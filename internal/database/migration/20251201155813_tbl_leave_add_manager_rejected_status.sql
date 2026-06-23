-- +goose Up
-- Drop existing constraint if it exists
ALTER TABLE Tbl_Leave DROP CONSTRAINT IF EXISTS chk_leave_status;

-- Add updated constraint with MANAGER_REJECTED status
ALTER TABLE Tbl_Leave 
ADD CONSTRAINT chk_leave_status 
CHECK (status IN ('Pending', 'MANAGER_APPROVED', 'MANAGER_REJECTED', 'APPROVED', 'REJECTED', 'CANCELLED', 'WITHDRAWAL_PENDING', 'WITHDRAWN'));

-- +goose Down
-- Revert to previous constraint (without MANAGER_REJECTED)
ALTER TABLE Tbl_Leave DROP CONSTRAINT IF EXISTS chk_leave_status;

ALTER TABLE Tbl_Leave 
ADD CONSTRAINT chk_leave_status 
CHECK (status IN ('Pending', 'MANAGER_APPROVED', 'APPROVED', 'REJECTED', 'CANCELLED', 'WITHDRAWAL_PENDING', 'WITHDRAWN'));
