-- +goose Up



CREATE TABLE IF NOT EXISTS Tbl_Company_Settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    working_days_per_month INT NOT NULL DEFAULT 22,
    allow_manager_add_leave BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    company_name VARCHAR(255) DEFAULT 'ZENITHIVE',
    logo_path TEXT DEFAULT '',
    primary_color VARCHAR(7) DEFAULT '#2980b9',
    secondary_color VARCHAR(7) DEFAULT '#2ecc71'
);

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
