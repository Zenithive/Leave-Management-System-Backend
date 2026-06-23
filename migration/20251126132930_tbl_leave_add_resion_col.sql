-- +goose Up
ALTER TABLE "tbl_leave"
ADD COLUMN IF NOT EXISTS reason TEXT DEFAULT '';

-- +goose Down
ALTER TABLE "tbl_leave"
DROP COLUMN IF EXISTS reason;