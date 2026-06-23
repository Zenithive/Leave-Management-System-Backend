-- +goose Up
-- +goose StatementBegin

-- 1️ Create leave timing master table
CREATE TABLE IF NOT EXISTS Tbl_Half (
    id INT PRIMARY KEY,
    type TEXT NOT NULL UNIQUE,        -- FIRST_HALF / SECOND_HALF / FULL
    timing TEXT NOT NULL,              -- e.g. 10:00-13:30 or FULL_DAY
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2️ Insert default timing values
INSERT INTO Tbl_Half (id, type, timing) VALUES
    (1, 'FIRST_HALF',  '10:00-13:30'),
    (2, 'SECOND_HALF', '13:30-19:00'),
    (3, 'FULL',        '10:00-19:00')
ON CONFLICT (id) DO NOTHING;

-- 3️ Add half_id column to leave table
ALTER TABLE Tbl_Leave
ADD COLUMN IF NOT EXISTS half_id INT DEFAULT 3;

-- 4️ Add foreign key constraint
ALTER TABLE Tbl_Leave
ADD CONSTRAINT fk_leave_half
FOREIGN KEY (half_id) REFERENCES Tbl_Half(id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

-- 1️ Drop foreign key constraint
ALTER TABLE Tbl_Leave
DROP CONSTRAINT IF EXISTS fk_leave_half;

-- 2️ Drop column
ALTER TABLE Tbl_Leave
DROP COLUMN IF EXISTS half_id;

-- 3️ Drop timing master table
DROP TABLE IF EXISTS Tbl_Half;

-- +goose StatementEnd
