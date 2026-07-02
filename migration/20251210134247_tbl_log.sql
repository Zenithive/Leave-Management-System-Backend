-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tbl_log(
    id SERIAL PRIMARY KEY,
    from_user_id UUID NOT NULL,
    action VARCHAR(255) NOT NULL,
    component VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tbl_log;
-- +goose StatementEnd
