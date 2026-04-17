package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// PayrollPreview represents preview data for a payroll run
type PayrollPreview struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	Employee     string    `json:"employee"`
	BasicSalary  float64   `json:"basic_salary"`
	WorkingDays  int       `json:"working_days"`
	PaidLeaves   float64   `json:"paid_leaves"`
	UnpaidLeaves float64   `json:"unpaid_leaves"`
	EarlyLeaves  float64   `json:"early_leaves"`
	Deductions   float64   `json:"deductions"`
	NetSalary    float64   `json:"net_salary"`
}

// RunPayroll handles payroll preview
func (h *HandlerFunc) RunPayroll(c *gin.Context) {
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "SUPERADMIN" && role != "ADMIN" {
		utils.RespondWithError(c, 403, "Not authorized to run payroll")
		return
	}

	var input struct {
		Month int `json:"month" validate:"required"`
		Year  int `json:"year" validate:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "Invalid input: "+err.Error())
		return
	}

	now := time.Now()
	if input.Year > now.Year() || (input.Year == now.Year() && input.Month >= int(now.Month())) {
		utils.RespondWithError(c, http.StatusBadRequest, "Cannot run payroll for future months")
		return
	}

	// --- Check if payroll already exists ---

	existingRun, err := h.Query.GetExitstingpayload(input)
	if err == nil {
		// Payroll run exists
		status := strings.ToUpper(strings.TrimSpace(existingRun.Status))

		if status == "FINALIZED" {
			utils.RespondWithError(c, 400, "Payroll for this month and year is already finalized. Cannot run payroll again.")
			return
		}
	}
	// --- Fetch active employees ---
	employees, err := h.Query.GetEmployeeByMonthAndYear(input)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to fetch employees: "+err.Error())
		return
	}
	// --- Fetch working days ---
	workingDays := h.Query.GetCompanyCurrWorkingDays()

	totalPayroll := 0.0
	totalDeductions := 0.0
	var previews []PayrollPreview

	for _, emp := range employees {
		// Set default salary to 0 if null
		salary := 0.0
		if emp.Salary != nil {
			salary = *emp.Salary
		}

		// Calculate absent days for this specific month only
		// Handle cross-month leaves correctly
		/*absentDays := service.CalculateAbsentDaysForMonth(h.Query.DB, emp.ID, input.Month, input.Year)
		if absentDays < 0 {
			utils.RespondWithError(c, 500, "Failed to calculate absent days")
			return
		}*/

		leaveSummary := service.CalculateAbsentDaysForMonth(h.Query.DB, emp.ID, input.Month, input.Year)

		deduction := salary / float64(workingDays) * leaveSummary.UnpaidDays
		net := salary - deduction

		previews = append(previews, PayrollPreview{
			EmployeeID:   emp.ID,
			Employee:     emp.FullName,
			BasicSalary:  salary,
			WorkingDays:  workingDays,
			PaidLeaves:   leaveSummary.PaidDays,
			UnpaidLeaves: leaveSummary.UnpaidDays,
			EarlyLeaves:  leaveSummary.EarlyLeaves,
			Deductions:   deduction,
			NetSalary:    net,
		})

		totalPayroll += net
		totalDeductions += deduction
	}

	// --- Create payroll run record ---
	runID := uuid.New()
	_, err = h.Query.DB.Exec(
		`INSERT INTO Tbl_Payroll_run (id, month, year, status) VALUES ($1,$2,$3,$4)`,
		runID, input.Month, input.Year, "PREVIEW",
	)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to create payroll run: "+err.Error())
		return
	}

	c.JSON(200, gin.H{
		"payroll_run_id":   runID,
		"month":            input.Month,
		"year":             input.Year,
		"total_payroll":    totalPayroll,
		"total_deductions": totalDeductions,
		"employees_count":  len(employees),
		"payroll_preview":  previews,
	})
}

// FinalizePayroll - generates payslips
// Only SUPERADMIN can finalize payroll
func (h *HandlerFunc) FinalizePayroll(c *gin.Context) {
	// --- Role Check - Only SUPERADMIN ---
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "SUPERADMIN" {
		utils.RespondWithError(c, 403, "Only SUPERADMIN can finalize payroll")
		return
	}

	// --- Parse Payroll Run ID ---
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid payroll run ID")
		return
	}

	// --- Fetch Payroll Run Data ---
	var run struct {
		Month  int    `db:"month"`
		Year   int    `db:"year"`
		Status string `db:"status"`
	}
	err = h.Query.DB.Get(&run,
		`SELECT month, year, status FROM Tbl_Payroll_run WHERE id=$1`, runID)
	if err != nil {
		utils.RespondWithError(c, 404, "Payroll run not found")
		return
	}

	// --- Block if Already Finalized ---
	if strings.ToUpper(strings.TrimSpace(run.Status)) == "FINALIZED" {
		utils.RespondWithError(c, 400, "Payroll is already finalized. Cannot finalize again. Finalized payrolls are locked and cannot be modified.")
		return
	}

	// --- Block if not in PREVIEW status ---
	if strings.ToUpper(strings.TrimSpace(run.Status)) != "PREVIEW" {
		utils.RespondWithError(c, 400, fmt.Sprintf("Cannot finalize payroll with status: %s. Only PREVIEW payrolls can be finalized.", run.Status))
		return
	}

	// --- Fetch working days ---
	var workingDays int
	err = h.Query.DB.Get(&workingDays,
		`SELECT working_days_per_month FROM Tbl_Company_Settings ORDER BY created_at DESC LIMIT 1`)
	if err != nil || workingDays <= 0 {
		workingDays = 22 // fallback default
	}

	// --- Transaction Start ---
	tx, err := h.Query.DB.Beginx()
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// --- Fetch Only Employees Belonging To The Payroll Run Period ---
	var employees []struct {
		ID       uuid.UUID `db:"id"`
		FullName string    `db:"full_name"`
		Salary   *float64  `db:"salary"`
	}

	err = tx.Select(&employees, `
        SELECT e.id, e.full_name, e.salary
        FROM Tbl_Employee e
        JOIN Tbl_Payroll_run r ON r.id = $1
        WHERE e.status='active'
          AND (
               EXTRACT(YEAR FROM e.joining_date) < r.year
               OR (EXTRACT(YEAR FROM e.joining_date) = r.year
                   AND EXTRACT(MONTH FROM e.joining_date) <= r.month)
              )
	`, runID)

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to fetch payroll employees: "+err.Error())
		return
	}

	// --- Generate Payslips ---
	var payslipIDs []uuid.UUID

	for _, emp := range employees {
		// Set default salary to 0 if null
		salary := 0.0
		if emp.Salary != nil {
			salary = *emp.Salary
		}

		// Calculate absent days for this specific month only
		// Handle cross-month leaves correctly
		/*
			absentDays := service.CalculateAbsentDaysForMonth(h.Query.DB, emp.ID, run.Month, run.Year)
			if absentDays < 0 {
				utils.RespondWithError(c, 500, "Failed to calculate absent days")
				return
			}*/
		leaveSummary := service.CalculateAbsentDaysForMonth(h.Query.DB, emp.ID, run.Month, run.Year)

		deduction := salary / float64(workingDays) * leaveSummary.UnpaidDays
		net := salary - deduction

		pID := uuid.New()
		_, err = tx.Exec(`
			INSERT INTO Tbl_Payslip 
			(id, payroll_run_id, employee_id, basic_salary, working_days, paid_leaves, unpaid_leaves, early_leaves, deduction_amount, net_salary)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, pID, runID, emp.ID, salary, workingDays, leaveSummary.PaidDays, leaveSummary.UnpaidDays, leaveSummary.EarlyLeaves, deduction, net)

		if err != nil {
			utils.RespondWithError(c, 500, "Payslip insert failed: "+err.Error())
			return
		}

		payslipIDs = append(payslipIDs, pID)
	}

	// --- Mark Payroll Run Finalized ---
	_, err = tx.Exec(`UPDATE Tbl_Payroll_run SET status='FINALIZED', updated_at=NOW() WHERE id=$1`, runID)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update payroll run: "+err.Error())
		return
	}

	if err = tx.Commit(); err != nil {
		utils.RespondWithError(c, 500, "Failed to commit: "+err.Error())
		return
	}

	// --- Success Response ---
	c.JSON(http.StatusOK, gin.H{
		"message":        "Payroll finalized successfully",
		"payroll_run_id": runID,
		"payslips":       payslipIDs,
	})
}

type lineItem struct {
	description string
	amount      float64
}

type PDFConfig struct {
	PrimaryColor []int
	CompanyName  string
	LogoPath     string
}

type PayslipReportData struct {
	EmployeeID   string
	EmployeeName string
	Email        string
	Month        string
	Year         int
	BasicSalary  float64
	WorkingDays  int
	PaidLeaves   float64
	UnpaidLeaves float64
	EarlyLeaves  float64
	Deductions   float64
	NetSalary    float64
}

// renderTable handles the repetitive task of drawing headers and rows

func renderTable(pdf *gofpdf.Fpdf, title string, items []lineItem, r, g, b int) float64 {

	// Section Header

	pdf.SetFont("Arial", "B", 14)

	pdf.SetFillColor(r, g, b)

	pdf.SetTextColor(255, 255, 255)

	pdf.CellFormat(0, 10, "  "+title, "", 1, "L", true, 0, "")

	// Column Headers

	pdf.SetTextColor(0, 0, 0)

	pdf.SetFont("Arial", "B", 11)

	pdf.SetFillColor(245, 245, 245)

	pdf.CellFormat(130, 9, "  Description", "1", 0, "L", true, 0, "")

	pdf.CellFormat(50, 9, "Amount (INR)", "1", 1, "C", true, 0, "")

	// Rows

	pdf.SetFont("Arial", "", 11)

	var total float64

	for _, item := range items {

		pdf.CellFormat(130, 9, "  "+item.description, "1", 0, "L", false, 0, "")

		pdf.CellFormat(50, 9, fmt.Sprintf("%.2f", item.amount), "1", 1, "R", false, 0, "")

		total += item.amount

	}

	// Total Row

	pdf.SetFont("Arial", "B", 11)

	pdf.SetFillColor(245, 245, 245)

	pdf.CellFormat(130, 9, "  TOTAL "+title, "1", 0, "L", true, 0, "")

	pdf.CellFormat(50, 9, fmt.Sprintf("%.2f", total), "1", 1, "R", true, 0, "")

	pdf.Ln(5)

	return total

}

func renderHeaderSection(pdf *gofpdf.Fpdf, d PayslipReportData, config PDFConfig) {
	// Top primary color bar
	pdf.SetFillColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
	pdf.Rect(0, 0, 210, 4, "F")

	// Handle Logo
	if config.LogoPath != "" {
		// We use a small offset from the top bar (10mm down)
		// Adjust the '30' to make logo bigger or smaller
		pdf.Image(config.LogoPath, 15, 10, 30, 0, false, "", 0, "")
	}

	// Set position for the Company Name and Statement
	// We start at Y=12 to align roughly with the top of the logo
	pdf.SetY(12)

	// Company Name
	pdf.SetFont("Arial", "B", 18) // Slightly smaller than 20 to look cleaner
	pdf.SetTextColor(50, 50, 50)
	pdf.CellFormat(0, 10, strings.ToUpper(config.CompanyName), "", 1, "R", false, 0, "")

	// Subtitle / Date
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 5, "Salary Statement: "+d.Month+" "+fmt.Sprint(d.Year), "", 1, "R", false, 0, "")

	// Vertical Spacer to move the cursor below the logo area before Employee Info starts
	pdf.SetY(45)
}

func renderEmployeeSection(pdf *gofpdf.Fpdf, d PayslipReportData, config PDFConfig) {
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
	pdf.CellFormat(0, 10, "  EMPLOYEE INFORMATION", "L", 1, "L", true, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(50, 50, 50)

	// Row 1 - Using CellFormat throughout for consistency
	pdf.CellFormat(30, 7, "Employee Name:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(70, 7, d.EmployeeName, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)

	// Row 2
	pdf.CellFormat(30, 7, "Email:", "", 0, "L", false, 0, "")
	pdf.CellFormat(70, 7, d.Email, "", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, "Working Days:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprint(d.WorkingDays), "", 1, "L", false, 0, "")
	pdf.Ln(8)
}

func renderSummarySection(pdf *gofpdf.Fpdf, d PayslipReportData, config PDFConfig) {
	pdf.SetX(130)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(35, 10, "NET PAYABLE:", "B", 0, "L", false, 0, "")

	pdf.SetTextColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(30, 10, "INR "+fmt.Sprintf("%.2f", d.NetSalary), "B", 1, "R", false, 0, "")
}

func renderFooterSection(pdf *gofpdf.Fpdf, d PayslipReportData) {
	pdf.SetY(270)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 5, "This is a computer-generated document and does not require a signature.", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("System Ref: %s | Date: %s", d.EmployeeID[:12], time.Now().Format("02-Jan-2006")), "", 1, "C", false, 0, "")
}
func renderAttendanceSummary(pdf *gofpdf.Fpdf, d PayslipReportData, config PDFConfig) {
	// Light gray background bar for the title, matching Employee Info style
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
	pdf.CellFormat(0, 10, "  ATTENDANCE & LEAVE SUMMARY", "L", 1, "L", true, 0, "")
	pdf.Ln(2)

	// Reset Text Color for the table
	pdf.SetTextColor(50, 50, 50)

	// Table Header
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(36, 8, "Working Days", "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, "Paid Leaves", "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, "Unpaid Leaves", "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, "Early Leaves", "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, "Days Present", "1", 1, "C", false, 0, "")

	// Table Data
	pdf.SetFont("Arial", "", 9)
	presentDays := float64(d.WorkingDays) - d.UnpaidLeaves

	pdf.CellFormat(36, 8, fmt.Sprintf("%d", d.WorkingDays), "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, fmt.Sprintf("%.1f", d.PaidLeaves), "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, fmt.Sprintf("%.1f", d.UnpaidLeaves), "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, fmt.Sprintf("%.1f", d.EarlyLeaves), "1", 0, "C", false, 0, "")
	pdf.CellFormat(36, 8, fmt.Sprintf("%.1f", presentDays), "1", 1, "C", false, 0, "")

	// Note about early leaves
	if d.EarlyLeaves > 0 {
		pdf.Ln(3)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 5, fmt.Sprintf("  * Early leaves (%.1f) are tracked for attendance only and do not affect salary deductions.", d.EarlyLeaves), "", 1, "L", false, 0, "")
		pdf.SetTextColor(50, 50, 50)
	}

	pdf.Ln(5)
}

/*
func renderAttendanceSummary(pdf *gofpdf.Fpdf, d PayslipReportData) {
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 10, "ATTENDANCE SUMMARY", "", 1, "L", false, 0, "")

	// Header
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(60, 8, "Total Working Days", "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, 8, "Days Present", "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, 8, "Days Absent", "1", 1, "C", false, 0, "")

	// Data
	pdf.SetFont("Arial", "", 9)
	presentDays := float64(d.WorkingDays) - d.UnpaidLeaves
	pdf.CellFormat(60, 8, fmt.Sprintf("%d days", d.WorkingDays), "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, 8, fmt.Sprintf("%.1f days", presentDays), "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, 8, fmt.Sprintf("%.1f days", d.UnpaidLeaves), "1", 1, "C", false, 0, "")
	pdf.Ln(5)
}*/

func (h *HandlerFunc) GetPayslipPDF(c *gin.Context) {
	payslipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid payslip ID")
		return
	}

	// 1. FETCH BRANDING
	var settings models.CompanySettings
	err = h.Query.DB.Get(&settings, `SELECT company_name, logo_path, primary_color FROM Tbl_Company_Settings LIMIT 1`)

	// If no settings found or name is the generic "Company", override it
	if err != nil || settings.CompanyName == "" || strings.EqualFold(settings.CompanyName, "Company") {
		settings.CompanyName = "ZENITHIVE"
	}

	r, g, b := HexToRGB(settings.PrimaryColor)
	config := PDFConfig{
		PrimaryColor: []int{r, g, b},
		CompanyName:  settings.CompanyName,
		LogoPath:     settings.LogoPath,
	}

	// 2. FETCH PAYSLIP DATA
	var p struct {
		EmployeeID   uuid.UUID `db:"employee_id"`
		EmployeeName string    `db:"full_name"`
		Email        string    `db:"email"`
		Month        int       `db:"month"`
		Year         int       `db:"year"`
		BasicSalary  float64   `db:"basic_salary"`
		WorkingDays  int       `db:"working_days"`
		PaidLeaves   float64   `db:"paid_leaves"`
		UnpaidLeaves float64   `db:"unpaid_leaves"`
		EarlyLeaves  float64   `db:"early_leaves"`
		Deductions   float64   `db:"deduction_amount"`
		NetSalary    float64   `db:"net_salary"`
	}

	err = h.Query.DB.Get(&p, `
        SELECT e.id as employee_id, e.full_name, e.email, p.basic_salary, 
               p.working_days, p.paid_leaves, p.unpaid_leaves, COALESCE(p.early_leaves, 0) as early_leaves,
               p.deduction_amount, p.net_salary,
               pr.month, pr.year
        FROM Tbl_Payslip p
        JOIN Tbl_Employee e ON e.id = p.employee_id
        JOIN Tbl_Payroll_run pr ON pr.id = p.payroll_run_id
        WHERE p.id = $1`, payslipID)

	if err != nil {
		utils.RespondWithError(c, 404, "Payslip not found")
		return
	}

	monthNames := []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	data := PayslipReportData{
		EmployeeID:   p.EmployeeID.String(),
		EmployeeName: p.EmployeeName,
		Email:        p.Email,
		Month:        monthNames[p.Month],
		Year:         p.Year,
		BasicSalary:  p.BasicSalary,
		WorkingDays:  p.WorkingDays,
		PaidLeaves:   p.PaidLeaves,
		UnpaidLeaves: p.UnpaidLeaves,
		EarlyLeaves:  p.EarlyLeaves,
		Deductions:   p.Deductions,
		NetSalary:    p.NetSalary,
	}

	// 3. GENERATE PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetAutoPageBreak(false, 0)

	renderHeaderSection(pdf, data, config)
	renderEmployeeSection(pdf, data, config)

	// Using the lineItem approach for the tables
	earnings := []lineItem{{description: "Basic Salary", amount: data.BasicSalary}}
	deductions := []lineItem{{description: fmt.Sprintf("Absent Leave (%v Days)", data.UnpaidLeaves), amount: data.Deductions}}

	renderTable(pdf, "EARNINGS", earnings, r, g, b)
	renderTable(pdf, "DEDUCTIONS", deductions, 231, 76, 60) // Red for deductions

	renderSummarySection(pdf, data, config)

	renderAttendanceSummary(pdf, data, config)
	renderFooterSection(pdf, data)

	// 4. SERVE FILE
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=payslip.pdf")
	pdf.Output(c.Writer)
}

// PreviewPayslipPDF - GET /api/payroll/payslips/preview
// Returns a sample payslip PDF with dummy data using real company branding.
// Only accessible by ADMIN and SUPERADMIN. No DB record is created.
func (h *HandlerFunc) PreviewPayslipPDF(c *gin.Context) {
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "SUPERADMIN" && role != "ADMIN" {
		utils.RespondWithError(c, 403, "Only ADMIN and SUPERADMIN can preview payslip")
		return
	}

	// 1. FETCH BRANDING (same as real payslip)
	var settings models.CompanySettings
	err := h.Query.DB.Get(&settings, `SELECT company_name, logo_path, primary_color FROM Tbl_Company_Settings LIMIT 1`)
	if err != nil || settings.CompanyName == "" || strings.EqualFold(settings.CompanyName, "Company") {
		settings.CompanyName = "ZENITHIVE"
	}

	r, g, b := HexToRGB(settings.PrimaryColor)
	config := PDFConfig{
		PrimaryColor: []int{r, g, b},
		CompanyName:  settings.CompanyName,
		LogoPath:     settings.LogoPath,
	}

	// 2. BUILD DUMMY DATA
	now := time.Now()
	workingDays := h.Query.GetCompanyCurrWorkingDays()
	basicSalary := 50000.0
	paidLeaves := 1.0
	unpaidLeaves := 2.0
	earlyLeaves := 1.0
	deduction := basicSalary / float64(workingDays) * unpaidLeaves
	netSalary := basicSalary - deduction

	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	data := PayslipReportData{
		EmployeeID:   "00000000-0000-0000-0000-000000000000",
		EmployeeName: "John Doe (Sample)",
		Email:        "john.doe@example.com",
		Month:        monthNames[int(now.Month())],
		Year:         now.Year(),
		BasicSalary:  basicSalary,
		WorkingDays:  workingDays,
		PaidLeaves:   paidLeaves,
		UnpaidLeaves: unpaidLeaves,
		EarlyLeaves:  earlyLeaves,
		Deductions:   deduction,
		NetSalary:    netSalary,
	}

	// 3. GENERATE PDF using the same render pipeline
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetAutoPageBreak(false, 0)

	renderHeaderSection(pdf, data, config)
	renderEmployeeSection(pdf, data, config)

	earnings := []lineItem{{description: "Basic Salary", amount: data.BasicSalary}}
	deductions := []lineItem{{description: fmt.Sprintf("Unpaid Leave (%v Days)", data.UnpaidLeaves), amount: data.Deductions}}

	renderTable(pdf, "EARNINGS", earnings, r, g, b)
	renderTable(pdf, "DEDUCTIONS", deductions, 231, 76, 60)

	renderSummarySection(pdf, data, config)
	renderAttendanceSummary(pdf, data, config)
	renderFooterSection(pdf, data)

	// 4. SERVE inline — no file saved, no DB record
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=payslip_preview.pdf")
	pdf.Output(c.Writer)
}

func HexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0 // Return black if invalid
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return int(r), int(g), int(b)
}

/*
// GetPayslipPDF - GET /api/payroll/payslips/:id/pdf
func (h *HandlerFunc) GetPayslipPDF(c *gin.Context) {
	payslipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid payslip ID"})
		return
	}

	var payslip struct {
		EmployeeID   uuid.UUID `db:"employee_id"`
		EmployeeName string    `db:"full_name"`
		Email        string    `db:"email"`
		Month        int       `db:"month"`
		Year         int       `db:"year"`
		BasicSalary  float64   `db:"basic_salary"`
		WorkingDays  int       `db:"working_days"`
		AbsentDays   float64   `db:"absent_days"`
		Deductions   float64   `db:"deduction_amount"`
		NetSalary    float64   `db:"net_salary"`
	}

	err = h.Query.DB.Get(&payslip, `
		SELECT e.id as employee_id, e.full_name, e.email,
		       p.basic_salary, p.working_days, p.absent_days,
		       p.deduction_amount, p.net_salary,
		       pr.month, pr.year
		FROM Tbl_Payslip p
		JOIN Tbl_Employee e ON e.id = p.employee_id
		JOIN Tbl_Payroll_run pr ON pr.id = p.payroll_run_id
		WHERE p.id = $1
	`, payslipID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Payslip not found: " + err.Error()})
		return
	}

	// Create PDF with improved design
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetAutoPageBreak(false, 0)

	// ========================================
	// HEADER SECTION - Company Branding
	// ========================================
	pdf.SetFillColor(41, 128, 185) // Professional blue
	pdf.Rect(0, 0, 210, 45, "F")

	pdf.SetTextColor(255, 255, 255) // White text
	pdf.SetFont("Arial", "B", 26)
	pdf.SetY(12)
	pdf.CellFormat(0, 10, "ZENITHIVE", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 12)
	pdf.SetY(25)
	pdf.CellFormat(0, 6, "Payroll Management System", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 14)
	pdf.SetY(35)
	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	pdf.CellFormat(0, 6, fmt.Sprintf("Salary Slip - %s %d", monthNames[payslip.Month], payslip.Year), "", 1, "C", false, 0, "")

	// ========================================
	// EMPLOYEE INFORMATION SECTION
	// ========================================
	pdf.SetY(55)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(236, 240, 241) // Light gray background
	pdf.CellFormat(0, 10, "  EMPLOYEE INFORMATION", "", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.Ln(2)

	// Employee details in two columns
	leftX := 15.0
	rightX := 110.0
	currentY := pdf.GetY()

	// Left column
	pdf.SetXY(leftX, currentY)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 7, "Employee Name:")
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, payslip.EmployeeName)

	currentY += 8
	pdf.SetXY(leftX, currentY)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 7, "Employee ID:")
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, payslip.EmployeeID.String()[:8]+"...")

	// Right column
	currentY = pdf.GetY() - 8
	pdf.SetXY(rightX, currentY)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(30, 7, "Email:")
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, payslip.Email)

	currentY += 8
	pdf.SetXY(rightX, currentY)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(30, 7, "Pay Period:")
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, fmt.Sprintf("%s %d", monthNames[payslip.Month], payslip.Year))

	// ========================================
	// EARNINGS SECTION
	// ========================================
	pdf.SetY(currentY + 15)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(46, 204, 113) // Green
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 10, "  EARNINGS", "", 1, "L", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(232, 245, 233) // Light green
	pdf.CellFormat(130, 9, "  Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 9, "Amount (INR)", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(130, 9, "  Basic Salary", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 9, fmt.Sprintf("%.2f", payslip.BasicSalary), "1", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(232, 245, 233)
	pdf.CellFormat(130, 9, "  GROSS EARNINGS", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 9, fmt.Sprintf("%.2f", payslip.BasicSalary), "1", 1, "R", true, 0, "")

	// ========================================
	// DEDUCTIONS SECTION
	// ========================================
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetFillColor(231, 76, 60) // Red
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 10, "  DEDUCTIONS", "", 1, "L", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(255, 235, 238) // Light red
	pdf.CellFormat(130, 9, "  Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 9, "Amount (INR)", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(130, 9, fmt.Sprintf("  Leave Deduction (%.1f absent days)", payslip.AbsentDays), "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 9, fmt.Sprintf("%.2f", payslip.Deductions), "1", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(255, 235, 238)
	pdf.CellFormat(130, 9, "  TOTAL DEDUCTIONS", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 9, fmt.Sprintf("%.2f", payslip.Deductions), "1", 1, "R", true, 0, "")

	// ========================================
	// NET SALARY SECTION (Highlighted)
	// ========================================
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 16)
	pdf.SetFillColor(52, 73, 94) // Dark blue
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(130, 12, "  NET SALARY (Take Home)", "1", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(50, 12, fmt.Sprintf("%.2f", payslip.NetSalary), "1", 1, "R", true, 0, "")

	// ========================================
	// ATTENDANCE SUMMARY
	// ========================================
	pdf.Ln(8)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(241, 196, 15) // Yellow/Gold
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 9, "  ATTENDANCE SUMMARY", "", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetFillColor(255, 249, 230) // Light yellow
	pdf.CellFormat(60, 8, "  Total Working Days", "1", 0, "L", true, 0, "")
	pdf.CellFormat(60, 8, "  Days Present", "1", 0, "L", true, 0, "")
	pdf.CellFormat(60, 8, "  Days Absent", "1", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(60, 8, fmt.Sprintf("  %d days", payslip.WorkingDays), "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 8, fmt.Sprintf("  %.1f days", float64(payslip.WorkingDays)-payslip.AbsentDays), "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 8, fmt.Sprintf("  %.1f days", payslip.AbsentDays), "1", 1, "L", false, 0, "")

	// ========================================
	// CALCULATION BREAKDOWN
	// ========================================
	pdf.Ln(8)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(189, 195, 199) // Gray
	pdf.CellFormat(0, 8, "  CALCULATION BREAKDOWN", "", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.Ln(2)
	pdf.MultiCell(0, 6, fmt.Sprintf(
		"Per Day Salary = Basic Salary / Working Days = %.2f / %d = %.2f\n"+
			"Leave Deduction = Per Day Salary x Absent Days = %.2f x %.1f = %.2f\n"+
			"Net Salary = Basic Salary - Leave Deduction = %.2f - %.2f = %.2f",
		payslip.BasicSalary, payslip.WorkingDays, payslip.BasicSalary/float64(payslip.WorkingDays),
		payslip.BasicSalary/float64(payslip.WorkingDays), payslip.AbsentDays, payslip.Deductions,
		payslip.BasicSalary, payslip.Deductions, payslip.NetSalary,
	), "", "L", false)

	// ========================================
	// FOOTER
	// ========================================
	pdf.SetY(270)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 5, "This is a computer-generated payslip and does not require a signature.", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated on: %s", time.Now().Format("02-Jan-2006 15:04:05")), "", 1, "C", false, 0, "")

	pdf.SetDrawColor(41, 128, 185)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, 285, 195, 285)

	// ========================================
	// SAVE PDF
	// ========================================
	os.MkdirAll("./tmp", os.ModePerm)
	pdfPath := fmt.Sprintf("./tmp/payslip_%s.pdf", payslipID)
	err = pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate PDF: " + err.Error()})
		return
	}

	// Update PDF path in DB
	_, _ = h.Query.DB.Exec("UPDATE Tbl_Payslip SET pdf_path=$1, updated_at=NOW() WHERE id=$2", pdfPath, payslipID)

	// Serve PDF
	c.File(pdfPath)
}
*/

func (h *HandlerFunc) GetFinalizedPayslips(c *gin.Context) {
	roleValue, ok := c.Get("role")
	if !ok {
		utils.RespondWithError(c, 500, "Failed to get role")
		return
	}
	role := roleValue.(string)

	var rows *sql.Rows
	var err error

	//  If Employee or Manager -> only their own slips
	if role == constant.ROLE_EMPLOYEE || role == constant.ROLE_MANAGER || role == constant.ROLE_HR {
		empIDValue, ok := c.Get("user_id")
		if !ok {
			utils.RespondWithError(c, 500, "Failed to get employee ID")
			return
		}

		// empIDValue is string, parse it to uuid
		empIDStr, ok := empIDValue.(string)
		if !ok {
			utils.RespondWithError(c, 500, "Invalid employee ID format")
			return
		}

		empID, err := uuid.Parse(empIDStr)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to parse employee ID: "+err.Error())
			return
		}
		rows, err = h.Query.GetFinalizedPayslipsByEmployee(empID)
	} else {
		//  SuperAdmin / Admin -> all slips
		rows, err = h.Query.GetAllFinalizedPayslips()
	}

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to fetch payslips: "+err.Error())
		return
	}

	// Only defer close if rows is not nil
	if rows != nil {
		defer rows.Close()
	} else {
		utils.RespondWithError(c, 500, "No rows returned from query")
		return
	}

	type FullPayslipResponse struct {
		PayslipID       uuid.UUID `json:"payslip_id"`
		EmployeeID      uuid.UUID `json:"employee_id"`
		FullName        string    `json:"full_name"`
		Email           string    `json:"email"`
		Month           int       `json:"month"`
		Year            int       `json:"year"`
		BasicSalary     float64   `json:"basic_salary"`
		WorkingDays     int       `json:"working_days"`
		PaidLeaves      float64   `json:"paid_leaves"`
		UnpaidLeaves    float64   `json:"unpaid_leaves"`
		EarlyLeaves     float64   `json:"early_leaves"`
		DeductionAmount float64   `json:"deduction_amount"`
		NetSalary       float64   `json:"net_salary"`
		PDFPath         string    `json:"pdf_path"`
		Calculation     string    `json:"calculation"`
		CreatedAt       string    `json:"created_at"`
	}

	var result []FullPayslipResponse

	for rows.Next() {
		var slip FullPayslipResponse
		err := rows.Scan(
			&slip.PayslipID,
			&slip.EmployeeID,
			&slip.FullName,
			&slip.Email,
			&slip.Month,
			&slip.Year,
			&slip.BasicSalary,
			&slip.WorkingDays,
			&slip.PaidLeaves,
			&slip.UnpaidLeaves,
			&slip.EarlyLeaves,
			&slip.DeductionAmount,
			&slip.NetSalary,
			&slip.PDFPath,
			&slip.Calculation,
			&slip.CreatedAt,
		)
		if err != nil {
			utils.RespondWithError(c, 500, "Scan failed: "+err.Error())
			return
		}
		result = append(result, slip)
	}

	if len(result) == 0 {
		c.JSON(200, gin.H{
			"message": "No finalized payslips found",
			"data":    []FullPayslipResponse{},
		})
		return
	}

	// 🎯 SUCCESS RESPONSE
	c.JSON(200, gin.H{
		"message":        "Finalized payslips fetched successfully",
		"total_payslips": len(result),
		"data":           result,
	})
}
