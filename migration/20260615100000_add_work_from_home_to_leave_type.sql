-- +goose Up
-- +goose StatementBegin

-- Add is_work_from_home flag to leave type policy.
-- When TRUE, employees on this leave type are considered working from home
-- (not absent). Defaults to FALSE for all existing records.
ALTER TABLE Tbl_Leave_type
ADD COLUMN IF NOT EXISTS is_work_from_home BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN Tbl_Leave_type.is_work_from_home IS
    'When true, this leave type is a work-from-home leave. Employees are not marked absent.';

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE Tbl_Leave_type
DROP COLUMN IF EXISTS is_work_from_home;

-- +goose StatementEnd
