-- =====================================================
-- COMPLETE DATABASE SCHEMA - HR & LEAVE MANAGEMENT SYSTEM
-- =====================================================
-- Database: PostgreSQL
-- Last Updated: April 9, 2026
-- Total Tables: 17
-- =====================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================
-- 1. TBL_ROLE - User Roles
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Role (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default roles
INSERT INTO Tbl_Role (type) VALUES 
    ('SUPERADMIN'),
    ('HR'),
    ('ADMIN'),
    ('MANAGER'),
    ('EMPLOYEE')
ON CONFLICT (type) DO NOTHING;

-- =====================================================
-- 2. TBL_DESIGNATION - Job Designations
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Designation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    designation_name VARCHAR(100) NOT NULL,
    description TEXT
);

-- =====================================================
-- 3. TBL_EMPLOYEE - Core Employee Information
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Employee (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    role_id INT NOT NULL REFERENCES Tbl_Role(id),
    password TEXT NOT NULL,
    manager_id UUID REFERENCES Tbl_Employee(id),  -- Self-referencing for hierarchy
    designation_id UUID REFERENCES Tbl_Designation(id) ON DELETE SET NULL,
    salary NUMERIC,
    birth_date DATE DEFAULT NULL,
    joining_date DATE,
    ending_date DATE DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',  -- active/deactive
    deleted_at TIMESTAMP NULL,  -- Soft delete
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 4. TBL_LEAVE_TYPE - Leave Type Master
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Leave_type (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    is_paid BOOLEAN,
    default_entitlement INT,
    is_early BOOLEAN DEFAULT FALSE,  -- Allows early leave (partial day)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 5. TBL_HALF - Leave Timing Master (Full/Half Day)
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Half (
    id INT PRIMARY KEY,
    type TEXT NOT NULL UNIQUE,  -- FIRST_HALF / SECOND_HALF / FULL
    timing TEXT NOT NULL,        -- e.g. 10:00-13:30 or FULL_DAY
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default timing values
INSERT INTO Tbl_Half (id, type, timing) VALUES
    (1, 'FIRST_HALF',  '10:00-13:30'),
    (2, 'SECOND_HALF', '13:30-19:00'),
    (3, 'FULL',        '10:00-19:00')
ON CONFLICT (id) DO NOTHING;

-- =====================================================
-- 6. TBL_LEAVE - Leave Applications
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Leave (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT NOT NULL REFERENCES Tbl_Leave_type(id),
    half_id INT DEFAULT 3 REFERENCES Tbl_Half(id),  -- Default: FULL day
    leave_timing TEXT DEFAULT NULL,  -- Optional timing for early leave
    start_date DATE,
    end_date DATE,
    days NUMERIC,
    status TEXT,
    applied_by UUID REFERENCES Tbl_Employee(id),  -- Who applied (can be manager)
    approved_by UUID REFERENCES Tbl_Employee(id),
    reason TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Status constraint
    CONSTRAINT chk_leave_status CHECK (
        status IN (
            'Pending', 
            'MANAGER_APPROVED', 
            'MANAGER_REJECTED', 
            'APPROVED', 
            'REJECTED', 
            'CANCELLED', 
            'WITHDRAWAL_PENDING', 
            'WITHDRAWN'
        )
    )
);

-- =====================================================
-- 7. TBL_LEAVE_BALANCE - Leave Balance Tracking
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Leave_balance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT NOT NULL REFERENCES Tbl_Leave_type(id),
    year INT,
    opening NUMERIC,   -- Opening balance
    accrued NUMERIC,   -- Accrued during year
    used NUMERIC,      -- Used leaves
    adjusted NUMERIC,  -- Manual adjustments
    closing NUMERIC,   -- Closing balance = opening + accrued - used + adjusted
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 8. TBL_LEAVE_ADJUSTMENT - Manual Leave Adjustments
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Leave_adjustment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    leave_type_id INT NOT NULL REFERENCES Tbl_Leave_type(id),
    quantity NUMERIC,
    reason TEXT,
    year INT,
    created_by UUID REFERENCES Tbl_Employee(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 9. TBL_HOLIDAY - Company Holidays
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Holiday (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    date DATE NOT NULL UNIQUE,
    day TEXT NOT NULL,
    type TEXT NOT NULL,  -- HOLIDAY or OPTIONAL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 10. TBL_PAYROLL_RUN - Monthly Payroll Processing
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Payroll_run (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month INT,
    year INT,
    status TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 11. TBL_PAYSLIP - Employee Payslips
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Payslip (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_run_id UUID NOT NULL REFERENCES Tbl_Payroll_run(id),
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    basic_salary NUMERIC,
    working_days INT,
    unpaid_leaves NUMERIC,  -- Changed from INT to NUMERIC for half-day support
    paid_leaves INT,
    early_leaves NUMERIC DEFAULT 0,  -- Early leave count (display only, no deduction)
    deduction_amount NUMERIC,
    net_salary NUMERIC,
    pdf_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- One payslip per employee per payroll run
    CONSTRAINT uq_payroll_employee UNIQUE (payroll_run_id, employee_id)
);

-- =====================================================
-- 12. TBL_EQUIPMENT_CATEGORY - Equipment Categories
-- =====================================================
CREATE TABLE IF NOT EXISTS tbl_equipment_category (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- =====================================================
-- 13. TBL_EQUIPMENT - Equipment Inventory
-- =====================================================
CREATE TABLE IF NOT EXISTS tbl_equipment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    category_id UUID NOT NULL REFERENCES tbl_equipment_category(id) ON DELETE RESTRICT,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,  -- Can multiple employees use?
    purchase_date DATE,
    price NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (price >= 0),
    total_quantity INT NOT NULL CHECK (total_quantity >= 0),
    remaining_quantity INT NOT NULL CHECK (
        remaining_quantity >= 0 AND 
        remaining_quantity <= total_quantity
    ),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- =====================================================
-- 14. TBL_EQUIPMENT_ASSIGNMENT - Equipment Assignment History
-- =====================================================
CREATE TABLE IF NOT EXISTS tbl_equipment_assignment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipment_id UUID NOT NULL REFERENCES tbl_equipment(id) ON DELETE RESTRICT,
    employee_id UUID NOT NULL REFERENCES Tbl_Employee(id) ON DELETE RESTRICT,
    assigned_by UUID NOT NULL REFERENCES Tbl_Employee(id) ON DELETE RESTRICT,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    assigned_at TIMESTAMP NOT NULL DEFAULT now(),
    returned_at TIMESTAMP  -- NULL = still assigned
);

-- =====================================================
-- 15. TBL_COMPANY_SETTINGS - Global Company Configuration
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Company_Settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    working_days_per_month INT NOT NULL DEFAULT 22,
    allow_manager_add_leave BOOLEAN NOT NULL DEFAULT FALSE,
    company_name VARCHAR(255) DEFAULT 'ZENITHIVE',
    logo_path TEXT DEFAULT '',
    primary_color VARCHAR(7) DEFAULT '#2980b9',
    secondary_color VARCHAR(7) DEFAULT '#2ecc71',
    -- Supports placeholders: {name}, {date}, {age}
    birthday_message_template TEXT NOT NULL DEFAULT 'Happy Birthday {name}! 🎉 Wishing you a wonderful day and a fantastic year ahead!',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings
INSERT INTO Tbl_Company_Settings (id, working_days_per_month, allow_manager_add_leave, company_name, primary_color, secondary_color)
VALUES (gen_random_uuid(), 22, false, 'ZENITHIVE', '#2980b9', '#2ecc71')
ON CONFLICT DO NOTHING;

-- =====================================================
-- 16. TBL_LOG - System Activity Logging
-- =====================================================
CREATE TABLE IF NOT EXISTS tbl_log (
    id SERIAL PRIMARY KEY,
    from_user_id UUID NOT NULL,  -- No FK for flexibility
    action VARCHAR(255) NOT NULL,
    component VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- 17. TBL_AUDIT - Detailed Audit Trail
-- =====================================================
CREATE TABLE IF NOT EXISTS Tbl_Audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL REFERENCES Tbl_Employee(id),
    action TEXT,
    entity TEXT,
    entity_id UUID,
    metadata JSONB,  -- JSON data for additional information
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Employee indexes
CREATE INDEX IF NOT EXISTS idx_employee_email ON Tbl_Employee(email);
CREATE INDEX IF NOT EXISTS idx_employee_manager ON Tbl_Employee(manager_id);
CREATE INDEX IF NOT EXISTS idx_employee_role ON Tbl_Employee(role_id);
CREATE INDEX IF NOT EXISTS idx_employee_designation ON Tbl_Employee(designation_id);
CREATE INDEX IF NOT EXISTS idx_employee_status ON Tbl_Employee(status);

-- Leave indexes
CREATE INDEX IF NOT EXISTS idx_leave_employee ON Tbl_Leave(employee_id);
CREATE INDEX IF NOT EXISTS idx_leave_status ON Tbl_Leave(status);
CREATE INDEX IF NOT EXISTS idx_leave_dates ON Tbl_Leave(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_leave_type ON Tbl_Leave(leave_type_id);

-- Leave balance indexes
CREATE INDEX IF NOT EXISTS idx_leave_balance_employee ON Tbl_Leave_balance(employee_id);
CREATE INDEX IF NOT EXISTS idx_leave_balance_year ON Tbl_Leave_balance(year);

-- Payslip indexes
CREATE INDEX IF NOT EXISTS idx_payslip_employee ON Tbl_Payslip(employee_id);
CREATE INDEX IF NOT EXISTS idx_payslip_payroll ON Tbl_Payslip(payroll_run_id);

-- Equipment indexes
CREATE INDEX IF NOT EXISTS idx_equipment_category ON tbl_equipment(category_id);
CREATE INDEX IF NOT EXISTS idx_equipment_assignment_employee ON tbl_equipment_assignment(employee_id);
CREATE INDEX IF NOT EXISTS idx_equipment_assignment_equipment ON tbl_equipment_assignment(equipment_id);

-- Log indexes
CREATE INDEX IF NOT EXISTS idx_log_user ON tbl_log(from_user_id);
CREATE INDEX IF NOT EXISTS idx_log_created ON tbl_log(created_at);

-- Audit indexes
CREATE INDEX IF NOT EXISTS idx_audit_actor ON Tbl_Audit(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON Tbl_Audit(entity, entity_id);

-- =====================================================
-- COMMENTS ON TABLES
-- =====================================================

COMMENT ON TABLE Tbl_Role IS 'User roles: SUPERADMIN, HR, ADMIN, MANAGER, EMPLOYEE';
COMMENT ON TABLE Tbl_Employee IS 'Core employee information with hierarchical structure';
COMMENT ON TABLE Tbl_Designation IS 'Job designations/positions';
COMMENT ON TABLE Tbl_Leave_type IS 'Leave type definitions with paid/unpaid and early leave support';
COMMENT ON TABLE Tbl_Half IS 'Leave timing master: FIRST_HALF, SECOND_HALF, FULL';
COMMENT ON TABLE Tbl_Leave IS 'Leave applications with approval workflow';
COMMENT ON TABLE Tbl_Leave_balance IS 'Annual leave balance tracking per employee';
COMMENT ON TABLE Tbl_Leave_adjustment IS 'Manual leave balance adjustments';
COMMENT ON TABLE Tbl_Holiday IS 'Company holidays calendar';
COMMENT ON TABLE Tbl_Payroll_run IS 'Monthly payroll processing runs';
COMMENT ON TABLE Tbl_Payslip IS 'Individual employee payslips with leave deductions';
COMMENT ON TABLE tbl_equipment_category IS 'Equipment categories';
COMMENT ON TABLE tbl_equipment IS 'Equipment inventory with quantity tracking';
COMMENT ON TABLE tbl_equipment_assignment IS 'Equipment assignment history';
COMMENT ON TABLE Tbl_Company_Settings IS 'Global company configuration and branding';
COMMENT ON TABLE tbl_log IS 'System activity logging';
COMMENT ON TABLE Tbl_Audit IS 'Detailed audit trail with JSON metadata';

-- =====================================================
-- END OF SCHEMA
-- =====================================================
