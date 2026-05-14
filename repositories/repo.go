package repositories

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type EmployeeAuthData struct {
	ID       string `db:"id"`
	Email    string `db:"email"`
	Password string `db:"password"`
	Role     string `db:"role"`
	Status   string `db:"status"`
}

type Repository struct {
	DB *sqlx.DB
}

func InitializeRepo(db *sqlx.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) GetAllFinalizedPayslips() (*sql.Rows, error) {
	return r.DB.Query(`
		SELECT
			p.id AS payslip_id,
			e.id AS employee_id,
			e.full_name,
			e.email,
			pr.month,
			pr.year,
			p.basic_salary,
			p.working_days,
			p.paid_leaves,
			p.unpaid_leaves,
			COALESCE(p.early_leaves, 0) AS early_leaves,
			p.deduction_amount,
			p.net_salary,
			COALESCE(p.pdf_path, '') AS pdf_path,
			CONCAT('₹', p.basic_salary, ' - ₹', p.deduction_amount, ' = ₹', p.net_salary) AS calculation,
			p.created_at
		FROM Tbl_Payslip p
		JOIN Tbl_Employee e ON p.employee_id = e.id
		JOIN Tbl_Payroll_Run pr ON pr.id = p.payroll_run_id
		WHERE pr.status = 'FINALIZED'
		ORDER BY pr.year DESC, pr.month DESC, e.full_name ASC
	`)
}

func (r *Repository) GetFinalizedPayslipsByEmployee(id uuid.UUID) (*sql.Rows, error) {
	return r.DB.Query(`
		SELECT
			p.id AS payslip_id,
			e.id AS employee_id,
			e.full_name,
			e.email,
			pr.month,
			pr.year,
			p.basic_salary,
			p.working_days,
			p.paid_leaves,
			p.unpaid_leaves,
			COALESCE(p.early_leaves, 0) AS early_leaves,
			p.deduction_amount,
			p.net_salary,
			COALESCE(p.pdf_path, '') AS pdf_path,
			CONCAT('₹', p.basic_salary, ' - ₹', p.deduction_amount, ' = ₹', p.net_salary) AS calculation,
			p.created_at
		FROM Tbl_Payslip p
		JOIN Tbl_Employee e ON p.employee_id = e.id
		JOIN Tbl_Payroll_Run pr ON pr.id = p.payroll_run_id
		WHERE pr.status = 'FINALIZED' AND e.id = $1
		ORDER BY pr.year DESC, pr.month DESC
	`, id)
}
