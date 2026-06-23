-- +goose Up

ALTER TABLE Tbl_Leave_adjustment
ADD COLUMN year INT;



-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
