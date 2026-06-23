-- +goose Up
-- +goose StatementBegin

-- 1️⃣ Add columns to Tbl_Leave table
-- Add is_early column to track if this is an early leave (default FALSE)

-- Add leave_timing column for optional timing information (default NULL)
ALTER TABLE Tbl_Leave
ADD COLUMN IF NOT EXISTS leave_timing TEXT DEFAULT NULL;

-- Add comments for documentation

COMMENT ON COLUMN Tbl_Leave.leave_timing IS 'Optional timing information for early leave (e.g., "14:00-17:00")';

-- 2️ Add is_early column to Tbl_Leave_type (leave policy) table (default FALSE)
ALTER TABLE Tbl_Leave_type
ADD COLUMN IF NOT EXISTS is_early BOOLEAN DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN Tbl_Leave_type.is_early IS 'Indicates if this leave type allows early leave requests';

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

-- Drop columns from Tbl_Leave_type
ALTER TABLE Tbl_Leave_type
DROP COLUMN IF EXISTS is_early;

-- Drop columns from Tbl_Leave
ALTER TABLE Tbl_Leave
DROP COLUMN IF EXISTS leave_timing;



-- +goose StatementEnd
