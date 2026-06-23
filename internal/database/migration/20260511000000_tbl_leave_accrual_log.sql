-- +goose Up
-- +goose StatementBegin

-- Tracks which month/year accrual has already been credited for each
-- employee + leave_type combination. This makes the accrual job idempotent:
-- if the cron fires twice in the same month, the second run is a no-op.
CREATE TABLE IF NOT EXISTS Tbl_Leave_accrual_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id   UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT  NOT NULL REFERENCES Tbl_Leave_type(id),
    month         INT  NOT NULL,   -- 1-12
    year          INT  NOT NULL,
    days_credited NUMERIC NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- One row per employee + leave_type + month + year
    CONSTRAINT uq_accrual_log UNIQUE (employee_id, leave_type_id, month, year)
);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS Tbl_Leave_accrual_log;

-- +goose StatementEnd
