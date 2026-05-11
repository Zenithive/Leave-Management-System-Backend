-- +goose Up
-- +goose StatementBegin

-- Ensures there is at most one balance row per employee + leave_type + year.
-- Required for the ON CONFLICT upsert in the monthly accrual job.
-- The DO NOTHING path in CreateLeaveBalance already relies on this being safe,
-- so adding the constraint makes that guarantee explicit at the DB level.
ALTER TABLE Tbl_Leave_balance
ADD CONSTRAINT uq_leave_balance UNIQUE (employee_id, leave_type_id, year);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE Tbl_Leave_balance
DROP CONSTRAINT IF EXISTS uq_leave_balance;

-- +goose StatementEnd
