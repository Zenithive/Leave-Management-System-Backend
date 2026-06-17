-- +goose Up

CREATE TABLE leave_approval_flow (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT NOT NULL,

    flow JSONB NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    deleted_at TIMESTAMP NULL
);
-- +goose Down

DROP TABLE IF EXISTS leave_approval_flow;