-- +goose Up
INSERT INTO Tbl_Company_Settings (id, working_days_per_month, allow_manager_add_leave)
VALUES (gen_random_uuid(), 22, false);
ALTER TABLE Tbl_Payslip
ADD CONSTRAINT uq_payroll_employee UNIQUE (payroll_run_id, employee_id);
-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
