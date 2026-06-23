-- +goose Up
-- Add ADMIN_APPROVED and ADMIN_REJECTED statuses for the two-stage approval
-- flow on Default and WorkFromHome leave types
-- (Admin/HR does first approval → SuperAdmin does final approval)

ALTER TABLE Tbl_Leave DROP CONSTRAINT IF EXISTS chk_leave_status;

ALTER TABLE Tbl_Leave
ADD CONSTRAINT chk_leave_status
CHECK (status IN (
    'Pending',
    'MANAGER_APPROVED',
    'MANAGER_REJECTED',
    'ADMIN_APPROVED',
    'ADMIN_REJECTED',
    'APPROVED',
    'REJECTED',
    'CANCELLED',
    'WITHDRAWAL_PENDING',
    'WITHDRAWN'
));

-- +goose Down
ALTER TABLE Tbl_Leave DROP CONSTRAINT IF EXISTS chk_leave_status;

ALTER TABLE Tbl_Leave
ADD CONSTRAINT chk_leave_status
CHECK (status IN (
    'Pending',
    'MANAGER_APPROVED',
    'MANAGER_REJECTED',
    'APPROVED',
    'REJECTED',
    'CANCELLED',
    'WITHDRAWAL_PENDING',
    'WITHDRAWN'
));
