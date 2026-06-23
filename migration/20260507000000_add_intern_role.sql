-- +goose Up
INSERT INTO Tbl_Role (type) VALUES ('INTERN') ON CONFLICT (type) DO NOTHING;

-- +goose Down
DELETE FROM Tbl_Role WHERE type = 'INTERN';
