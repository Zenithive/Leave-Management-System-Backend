package repositories

import (
	"strings"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

func buildReportPeriodCTE() string {
	return `
WITH report_period AS (
	SELECT
		DATE_TRUNC('month', MAKE_DATE($1::int, $2::int, 1))::date AS win_start,

		(
			DATE_TRUNC('month', MAKE_DATE($3::int, $4::int, 1))
			+ INTERVAL '1 month - 1 day'
		)::date AS win_end
)
`
}

func buildLeaveWorkingDaysCTE() string {
	return `
, leave_working_days AS (
	SELECT
		l.id AS leave_id,
		l.employee_id,
		l.days,
		lt.is_paid,
		lt.is_early,

		(
			SELECT COUNT(*)
			FROM generate_series(
				l.start_date,
				l.end_date,
				INTERVAL '1 day'
			) d
			WHERE EXTRACT(DOW FROM d) NOT IN (0, 6)
		) AS total_wd,

		(
			SELECT COUNT(*)
			FROM generate_series(
				GREATEST(l.start_date, rp.win_start),
				LEAST(l.end_date, rp.win_end),
				INTERVAL '1 day'
			) d
			WHERE EXTRACT(DOW FROM d) NOT IN (0, 6)
		) AS overlap_wd

	FROM Tbl_Leave l

	JOIN Tbl_Leave_Type lt
		ON lt.id = l.leave_type_id

	CROSS JOIN report_period rp

	WHERE l.status = 'APPROVED'
	  AND l.start_date <= rp.win_end
	  AND l.end_date >= rp.win_start
)
`
}

func buildProratedLeavesCTE() string {
	return `
, prorated_leaves AS (
	SELECT
		employee_id,
		is_paid,
		is_early,

		ROUND(
			days::numeric
			* overlap_wd::numeric
			/ NULLIF(total_wd::numeric, 0),
			2
		) AS prorated_days

	FROM leave_working_days

	WHERE overlap_wd > 0
)
`
}

func buildBalanceSummaryCTE() string {
	return `
, balance_summary AS (
	SELECT
		lb.employee_id,

		COALESCE(
			SUM(lb.closing),
			0
		) AS balance_leaves

	FROM Tbl_Leave_balance lb

	CROSS JOIN report_period rp

	WHERE lb.year BETWEEN
		EXTRACT(YEAR FROM rp.win_start)::int
		AND EXTRACT(YEAR FROM rp.win_end)::int

	GROUP BY lb.employee_id
)
`
}

func buildAccrualSummaryCTE() string {
	return `
, accrual_summary AS (
	SELECT
		a.employee_id,

		COALESCE(
			SUM(a.days_credited),
			0
		) AS accrued_leaves

	FROM Tbl_Leave_accrual_log a

	CROSS JOIN report_period rp

	WHERE (
		a.year > EXTRACT(YEAR FROM rp.win_start)::int

		OR (
			a.year = EXTRACT(YEAR FROM rp.win_start)::int
			AND a.month >= EXTRACT(MONTH FROM rp.win_start)::int
		)
	)

	AND (
		a.year < EXTRACT(YEAR FROM rp.win_end)::int

		OR (
			a.year = EXTRACT(YEAR FROM rp.win_end)::int
			AND a.month <= EXTRACT(MONTH FROM rp.win_end)::int
		)
	)

	GROUP BY a.employee_id
)
`
}

func buildFinalLeaveReportSelect() string {
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
				AND (
					pl.is_early IS NULL
					OR pl.is_early = FALSE
				)
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
				AND (
					pl.is_early IS NULL
					OR pl.is_early = FALSE
				)
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS unpaid_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN pl.is_early = TRUE
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS early_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN (
					pl.is_early IS NULL
					OR pl.is_early = FALSE
				)
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS used_leaves,

	COALESCE(
		SUM(
			CASE
				WHEN (
					pl.is_early IS NULL
					OR pl.is_early = FALSE
				)
				THEN pl.prorated_days
				ELSE 0
			END
		),
		0
	) AS total_leaves

FROM Tbl_Employee e

JOIN Tbl_Role r
	ON e.role_id = r.id

LEFT JOIN prorated_leaves pl
	ON pl.employee_id = e.id

LEFT JOIN balance_summary bs
	ON bs.employee_id = e.id

LEFT JOIN accrual_summary ac
	ON ac.employee_id = e.id

WHERE e.status = 'active'
  AND r.type != 'SUPERADMIN'

  AND (
	$5 = ''
	OR LOWER(e.full_name) LIKE '%' || LOWER($5) || '%'
	OR LOWER(e.email) LIKE '%' || LOWER($5) || '%'
  )

  AND (
	$6 = ''
	OR UPPER(r.type) = UPPER($6)
  )

GROUP BY
	e.id,
	e.full_name,
	e.email,
	r.type,
	ac.accrued_leaves,
	bs.balance_leaves
`
}

func (r *Repository) GetLeaveReportByRange(
	filter models.LeaveReportFilter,
) ([]models.LeaveReportRecord, error) {

	query := strings.Join([]string{
		buildReportPeriodCTE(),
		buildLeaveWorkingDaysCTE(),
		buildProratedLeavesCTE(),
		buildBalanceSummaryCTE(),
		buildAccrualSummaryCTE(),
		buildFinalLeaveReportSelect(),
	}, "\n")

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
