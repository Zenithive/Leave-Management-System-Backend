-- +goose Up

CREATE TABLE IF NOT EXISTS tbl_leave_flow (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    leave_id UUID NOT NULL,

    approval_log JSONB NOT NULL DEFAULT '[]'::jsonb,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,

    CONSTRAINT fk_leave_flow_leave
        FOREIGN KEY (leave_id)
        REFERENCES tbl_leave(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_leave_flow_leave_id
ON tbl_leave_flow(leave_id);

CREATE INDEX idx_leave_flow_deleted_at
ON tbl_leave_flow(deleted_at);

CREATE INDEX idx_leave_flow_approval_log
ON tbl_leave_flow USING GIN (approval_log);

-- +goose Down

DROP TABLE IF EXISTS tbl_leave_flow;