-- +goose Up
INSERT INTO Tbl_Company_Settings (id, working_days_per_month, allow_manager_add_leave,company_name, primary_color, secondary_color)
VALUES (gen_random_uuid(), 22, false,'ZENITHIVE', '#2980b9', '#2ecc71')ON CONFLICT DO NOTHING;


-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
