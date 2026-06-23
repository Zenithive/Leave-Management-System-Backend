-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
INSERT INTO Tbl_Company_Settings (id, working_days_per_month, allow_manager_add_leave)
VALUES (gen_random_uuid(), 22, false);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
