-- +goose Up

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- 1. Roles
CREATE TABLE IF NOT EXISTS Tbl_Role (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Employees
CREATE TABLE IF NOT EXISTS Tbl_Employee (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    role_id INT NOT NULL REFERENCES Tbl_Role(id),
    password TEXT NOT NULL,
    manager_id UUID REFERENCES Tbl_Employee(id),
    salary NUMERIC,
    joining_date DATE,
    deleted_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Leave Types
CREATE TABLE IF NOT EXISTS Tbl_Leave_type (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    is_paid BOOLEAN,
    default_entitlement INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. Leaves
CREATE TABLE IF NOT EXISTS Tbl_Leave (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT NOT NULL REFERENCES Tbl_Leave_type(id),
    start_date DATE,
    end_date DATE,
    days NUMERIC,
    status TEXT,
    applied_by UUID REFERENCES Tbl_Employee(id),
    approved_by UUID REFERENCES Tbl_Employee(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. Leave Balances
CREATE TABLE IF NOT EXISTS Tbl_Leave_balance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT NOT NULL REFERENCES Tbl_Leave_type(id),
    year INT,
    opening NUMERIC,
    accrued NUMERIC,
    used NUMERIC,
    adjusted NUMERIC,
    closing NUMERIC,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. Leave Adjustments
CREATE TABLE IF NOT EXISTS Tbl_Leave_adjustment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT NOT NULL REFERENCES Tbl_Leave_type(id),
    quantity NUMERIC,
    reason TEXT,
    created_by UUID REFERENCES Tbl_Employee(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. Payroll Runs
CREATE TABLE IF NOT EXISTS Tbl_Payroll_run (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month INT,
    year INT,
    status TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 8. Payslips
CREATE TABLE IF NOT EXISTS Tbl_Payslip (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_run_id UUID NOT NULL REFERENCES Tbl_Payroll_run(id),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    basic_salary NUMERIC,
    working_days INT,
    absent_days INT,
    deduction_amount NUMERIC,
    net_salary NUMERIC,
    pdf_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 9. Audits
CREATE TABLE IF NOT EXISTS Tbl_Audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    action TEXT,
    entity TEXT,
    entity_id UUID,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS Tbl_Holiday (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,                    
    date DATE NOT NULL UNIQUE,              
    day TEXT NOT NULL,                     
    type TEXT NOT NULL,                     
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE tbl_employee
ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE Tbl_Role


ADD CONSTRAINT uq_role_type UNIQUE (type);
 INSERT INTO Tbl_Role (type) VALUES ('SUPERADMIN') ON CONFLICT (type) DO NOTHING;
 INSERT INTO Tbl_Role (type) VALUES ('HR') ON CONFLICT (type) DO NOTHING;
INSERT INTO Tbl_Role (type) VALUES ('ADMIN') ON CONFLICT (type) DO NOTHING;
INSERT INTO Tbl_Role (type) VALUES ('MANAGAR') ON CONFLICT (type) DO NOTHING;
INSERT INTO Tbl_Role (type) VALUES ('EMPLOYEE') ON CONFLICT (type) DO NOTHING;


-- +goose Down


DROP TABLE IF EXISTS Tbl_Audit;
DROP TABLE IF EXISTS Tbl_Payslip;
DROP TABLE IF EXISTS Tbl_Payroll_run;
DROP TABLE IF EXISTS Tbl_Leave_adjustment;
DROP TABLE IF EXISTS Tbl_Leave_balance;
DROP TABLE IF EXISTS Tbl_Leave;
DROP TABLE IF EXISTS Tbl_Leave_type;
DROP TABLE IF EXISTS Tbl_Employee;
DROP TABLE IF EXISTS Tbl_Role;


