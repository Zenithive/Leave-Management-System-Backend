-- +goose Up
-- +goose StatementBegin

ALTER TABLE Tbl_Payslip
ADD COLUMN IF NOT EXISTS early_leaves NUMERIC DEFAULT 0;

COMMENT ON COLUMN Tbl_Payslip.early_leaves IS 'Count of early leave days in the pay period — tracked for attendance display only, does not affect salary deduction';

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE Tbl_Payslip
DROP COLUMN IF EXISTS early_leaves;

-- +goose StatementEnd
