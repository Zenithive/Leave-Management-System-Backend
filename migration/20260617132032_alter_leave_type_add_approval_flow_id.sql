-- +goose Up
ALTER TABLE Tbl_Leave_type
ADD COLUMN approval_flow_id UUID;

ALTER TABLE Tbl_Leave_type
ADD CONSTRAINT fk_leave_type_approval_flow
FOREIGN KEY (approval_flow_id)
REFERENCES leave_approval_flow(id)
ON DELETE SET NULL;

CREATE INDEX idx_leave_type_approval_flow_id
ON Tbl_Leave_type(approval_flow_id);

-- +goose Down
DROP INDEX IF EXISTS idx_leave_type_approval_flow_id;

ALTER TABLE Tbl_Leave_type
DROP CONSTRAINT IF EXISTS fk_leave_type_approval_flow;

ALTER TABLE Tbl_Leave_type
DROP COLUMN IF EXISTS approval_flow_id;