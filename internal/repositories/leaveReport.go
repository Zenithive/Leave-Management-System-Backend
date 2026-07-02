package repositories

import (
	"strings"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
)

// ============================================================================
// 1. FOUNDATION LAYER - Reusable Building Blocks
// ============================================================================

type SQLBuilder struct {
	parts []string
}

func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{parts: make([]string, 0)}
}

func (sb *SQLBuilder) Add(part string) *SQLBuilder {
	sb.parts = append(sb.parts, part)
	return sb
}

func (sb *SQLBuilder) Build() string {
	return strings.Join(sb.parts, "\n")
}

// ============================================================================
// 2. UTILITY LAYER - Common Date/Working Days Logic
// ============================================================================

type WorkingDaysCalculator struct{}

func (wdc *WorkingDaysCalculator) BuildHolidaysCTE() string {
	return `
WITH holidays AS (
	SELECT date::date AS holiday_date 
	FROM Tbl_Holiday
)`
}

func (wdc *WorkingDaysCalculator) BuildWorkingDaysFormula(
	startDate, endDate string,
) string {
	return `(
		SELECT COUNT(*)
		FROM generate_series(
			` + startDate + `,
			` + endDate + `,
			INTERVAL '1 day'
		) d
		WHERE EXTRACT(DOW FROM d) NOT IN (0, 6)
		  AND d::date NOT IN (
			SELECT holiday_date FROM holidays
			WHERE holiday_date BETWEEN ` + startDate + `::date AND ` + endDate + `::date
		  )
	)`
}

// ============================================================================
// 3. BUSINESS LOGIC LAYER - Leave Calculations
// ============================================================================

type LeaveCalculationCTEs struct {
	wdCalc *WorkingDaysCalculator
}

func NewLeaveCalculationCTEs() *LeaveCalculationCTEs {
	return &LeaveCalculationCTEs{
		wdCalc: &WorkingDaysCalculator{},
	}
}

func (lc *LeaveCalculationCTEs) BuildReportPeriodCTE() string {
	return `
, report_period AS (
	SELECT
		DATE_TRUNC('month', MAKE_DATE($1::int, $2::int, 1))::date AS win_start,
		(
			DATE_TRUNC('month', MAKE_DATE($3::int, $4::int, 1))
			+ INTERVAL '1 month - 1 day'
		)::date AS win_end
)`
}

func (lc *LeaveCalculationCTEs) BuildLeaveWorkingDaysCTE() string {
	totalWDFormula := lc.wdCalc.BuildWorkingDaysFormula(
		"l.start_date",
		"l.end_date",
	)

	overlapWDFormula := lc.wdCalc.BuildWorkingDaysFormula(
		"GREATEST(l.start_date, rp.win_start)",
		"LEAST(l.end_date, rp.win_end)",
	)

	return `
, leave_working_days AS (
	SELECT
		l.id AS leave_id,
		l.employee_id,
		l.days AS original_days,
		lt.is_paid,
		lt.is_early,

		` + totalWDFormula + ` AS total_wd,
		` + overlapWDFormula + ` AS overlap_wd

	FROM Tbl_Leave l
	JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
	CROSS JOIN report_period rp

	WHERE l.status = 'APPROVED'
	  AND l.start_date <= rp.win_end
	  AND l.end_date >= rp.win_start
)`
}

func (lc *LeaveCalculationCTEs) BuildProratedLeavesCTE() string {
	return `
, prorated_leaves AS (
	SELECT
		employee_id,
		is_paid,
		is_early,

		CASE
			WHEN overlap_wd <= 0 THEN 0
			WHEN total_wd = overlap_wd THEN original_days
			WHEN total_wd = 0 THEN 0
			ELSE (overlap_wd::numeric / total_wd) * original_days
		END AS prorated_days

	FROM leave_working_days
	WHERE overlap_wd > 0
)`
}

// ============================================================================
// 4. AGGREGATION LAYER - Balance & Accrual
// ============================================================================

type LeaveAggregationCTEs struct{}

func NewLeaveAggregationCTEs() *LeaveAggregationCTEs {
	return &LeaveAggregationCTEs{}
}

func (la *LeaveAggregationCTEs) BuildBalanceSummaryCTE() string {
	return `
, balance_summary AS (
	SELECT
		lb.employee_id,
		COALESCE(SUM(lb.closing), 0) AS balance_leaves

	FROM Tbl_Leave_balance lb
	CROSS JOIN report_period rp

	WHERE lb.year BETWEEN
		EXTRACT(YEAR FROM rp.win_start)::int
		AND EXTRACT(YEAR FROM rp.win_end)::int

	GROUP BY lb.employee_id
)`
}

func (la *LeaveAggregationCTEs) BuildAccrualSummaryCTE() string {
	return `
, accrual_summary AS (
	SELECT
		a.employee_id,
		COALESCE(SUM(a.days_credited), 0) AS accrued_leaves

	FROM Tbl_Leave_accrual_log a
	CROSS JOIN report_period rp

	WHERE (
		a.year > EXTRACT(YEAR FROM rp.win_start)::int
		AND a.year < EXTRACT(YEAR FROM rp.win_end)::int
	)
	OR (
		a.year = EXTRACT(YEAR FROM rp.win_start)::int
		AND a.month >= EXTRACT(MONTH FROM rp.win_start)::int
		AND (
			a.year < EXTRACT(YEAR FROM rp.win_end)::int
			OR a.month <= EXTRACT(MONTH FROM rp.win_end)::int
		)
	)
	OR (
		a.year = EXTRACT(YEAR FROM rp.win_end)::int
		AND a.month <= EXTRACT(MONTH FROM rp.win_end)::int
		AND (
			a.year > EXTRACT(YEAR FROM rp.win_start)::int
			OR a.month >= EXTRACT(MONTH FROM rp.win_start)::int
		)
	)

	GROUP BY a.employee_id
)`
}

// ============================================================================
// 5. PRESENTATION LAYER - Final Select
// ============================================================================

type LeaveReportPresentation struct{}

func NewLeaveReportPresentation() *LeaveReportPresentation {
	return &LeaveReportPresentation{}
}

func (lrp *LeaveReportPresentation) BuildFinalSelect() string {
	return `
SELECT
	e.id::text AS employee_id,
	e.full_name AS employee_name,
	e.email AS email,
	r.type AS role,

	COALESCE(ac.accrued_leaves, 0) AS accrued_leaves,
	COALESCE(bs.balance_leaves, 0) AS balance_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN pl.is_paid = TRUE
				AND COALESCE(pl.is_early, FALSE) = FALSE
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS paid_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN pl.is_paid = FALSE
				AND COALESCE(pl.is_early, FALSE) = FALSE
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS unpaid_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN COALESCE(pl.is_early, FALSE) = TRUE
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS early_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN COALESCE(pl.is_early, FALSE) = FALSE
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS used_leaves,

	COALESCE(
		SUM(pl.prorated_days),
		0
	) AS total_leaves

FROM Tbl_Employee e
JOIN Tbl_Role r ON e.role_id = r.id
LEFT JOIN prorated_leaves pl ON pl.employee_id = e.id
LEFT JOIN balance_summary bs ON bs.employee_id = e.id
LEFT JOIN accrual_summary ac ON ac.employee_id = e.id

WHERE e.status = 'active'
  AND r.type != 'SUPERADMIN'
  AND ($5 = '' OR LOWER(e.full_name) LIKE '%' || LOWER($5) || '%' OR LOWER(e.email) LIKE '%' || LOWER($5) || '%')
  AND ($6 = '' OR UPPER(r.type) = UPPER($6))

GROUP BY
	e.id,
	e.full_name,
	e.email,
	r.type,
	ac.accrued_leaves,
	bs.balance_leaves`
}

// ============================================================================
// 6. ORCHESTRATION LAYER - Query Builder
// ============================================================================

type LeaveReportQueryBuilder struct {
	wdCalc       *WorkingDaysCalculator
	leaveCalc    *LeaveCalculationCTEs
	aggregation  *LeaveAggregationCTEs
	presentation *LeaveReportPresentation
}

func NewLeaveReportQueryBuilder() *LeaveReportQueryBuilder {
	return &LeaveReportQueryBuilder{
		wdCalc:       &WorkingDaysCalculator{},
		leaveCalc:    NewLeaveCalculationCTEs(),
		aggregation:  NewLeaveAggregationCTEs(),
		presentation: NewLeaveReportPresentation(),
	}
}

func (qb *LeaveReportQueryBuilder) BuildFullQuery() string {
	builder := NewSQLBuilder()

	return builder.
		Add(qb.wdCalc.BuildHolidaysCTE()).
		Add(qb.leaveCalc.BuildReportPeriodCTE()).
		Add(qb.leaveCalc.BuildLeaveWorkingDaysCTE()).
		Add(qb.leaveCalc.BuildProratedLeavesCTE()).
		Add(qb.aggregation.BuildBalanceSummaryCTE()).
		Add(qb.aggregation.BuildAccrualSummaryCTE()).
		Add(qb.presentation.BuildFinalSelect()).
		Build()
}

// ============================================================================
// 7. REPOSITORY LAYER - Database Execution
// ============================================================================

func (r *Repository) GetLeaveReportByRange(
	filter models.LeaveReportFilter,
) ([]models.LeaveReportRecord, error) {

	queryBuilder := NewLeaveReportQueryBuilder()
	query := queryBuilder.BuildFullQuery()

	query += models.BuildLeaveReportOrder(
		filter.SortBy,
		filter.SortOrder,
	)

	var rows []models.LeaveReportRecord

	err := r.DB.Select(
		&rows,
		query,
		filter.FromYear,
		filter.FromMonth,
		filter.ToYear,
		filter.ToMonth,
		filter.Search,
		filter.Role,
	)

	if err != nil {
		return nil, err
	}

	return rows, nil
}
