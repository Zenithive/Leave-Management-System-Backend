-- +goose Up

CREATE TABLE leave_approval_flow (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT NOT NULL,

    is_system BOOLEAN DEFAULT false,

    flow JSONB NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO leave_approval_flow (
    name,
    is_system,
    flow
)
VALUES (
    'Default Flows',
    true,
    '[
        {"stage_no": 1, "approver_role": "MANAGER"},
        {"stage_no": 1, "approver_role": "ADMIN"},
        {"stage_no": 1, "approver_role": "HR"},
        {"stage_no": 2, "approver_role": "SUPERADMIN"}
    ]'::jsonb
);

-- +goose Down

DROP TABLE IF EXISTS leave_approval_flow;