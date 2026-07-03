-- +goose Up
-- +goose StatementBegin

-- ===============================
-- Equipment Category
-- ===============================
CREATE TABLE IF NOT EXISTS tbl_equipment_category (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT DEFAULT '',

    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ===============================
-- Equipment Master
-- ===============================
CREATE TABLE IF NOT EXISTS tbl_equipment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    category_id UUID NOT NULL
        REFERENCES tbl_equipment_category(id)
        ON DELETE RESTRICT,
    -- Can multiple employees use it at same time?
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    purchase_date DATE,          -- << COMMA ADDED HERE
    -- Price per unit
    price NUMERIC(12,2) NOT NULL DEFAULT 0
        CHECK (price >= 0),
    -- Total quantity owned
    total_quantity INT NOT NULL
        CHECK (total_quantity >= 0),
    -- Remaining available quantity
    remaining_quantity INT NOT NULL
        CHECK (
            remaining_quantity >= 0
            AND remaining_quantity <= total_quantity
        ),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
-- ===============================
-- Equipment Assignment
-- ===============================
CREATE TABLE IF NOT EXISTS tbl_equipment_assignment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    equipment_id UUID NOT NULL
        REFERENCES tbl_equipment(id)
        ON DELETE RESTRICT,

    -- Employee who receives equipment
    employee_id UUID NOT NULL
        REFERENCES tbl_employee(id)
        ON DELETE RESTRICT,

    -- Employee/Admin who assigned it
    assigned_by UUID NOT NULL
        REFERENCES tbl_employee(id)
        ON DELETE RESTRICT,

    quantity INT NOT NULL DEFAULT 1
        CHECK (quantity > 0),

    assigned_at TIMESTAMP NOT NULL DEFAULT now(),
    returned_at TIMESTAMP
);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tbl_equipment_assignment;
DROP TABLE IF EXISTS tbl_equipment;
DROP TABLE IF EXISTS tbl_equipment_category;
-- +goose StatementEnd
