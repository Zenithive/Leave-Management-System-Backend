-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE tbl_payslip 
RENAME COLUMN absent_days TO unpaid_leaves;

ALTER TABLE tbl_payslip 
ADD COLUMN paid_leaves INT;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE tbl_payslip 
DROP COLUMN paid_leaves;

ALTER TABLE tbl_payslip 
RENAME COLUMN unpaid_leaves TO absent_days;

-- +goose StatementEnd
