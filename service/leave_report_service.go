package service

import (
	"fmt"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
)

// LeaveReportService handles business logic for all leave report types.
type LeaveReportService struct {
	repo *repositories.Repository
}

// NewLeaveReportService creates a new LeaveReportService.
func NewLeaveReportService(repo *repositories.Repository) *LeaveReportService {
	return &LeaveReportService{repo: repo}
}

// GetLeaveReport resolves the date range from the request and delegates to the repository.
// Supported report types:
//   - "monthly"  → single month  (req.Month, req.Year)
//   - "yearly"   → full year     (Jan–Dec of req.Year)
//   - "range"    → custom range  (req.FromMonth/FromYear → req.ToMonth/ToYear)
func (s *LeaveReportService) GetLeaveReport(
	req models.LeaveReportRequest,
) (models.LeaveReportResponse, error) {

	var fromMonth, fromYear int
	var toMonth, toYear int

	switch req.ReportType {

	case "monthly":
		fromMonth = req.Month
		fromYear = req.Year

		toMonth = req.Month
		toYear = req.Year

	case "yearly":
		fromMonth = 1
		fromYear = req.Year

		toMonth = 12
		toYear = req.Year

	case "range":
		fromMonth = req.FromMonth
		fromYear = req.FromYear

		toMonth = req.ToMonth
		toYear = req.ToYear

		// Validate range order
		if fromYear*12+fromMonth > toYear*12+toMonth {
			return models.LeaveReportResponse{},
				fmt.Errorf("from date must be before or equal to to date")
		}

	default:
		return models.LeaveReportResponse{},
			fmt.Errorf(
				"invalid report_type: must be monthly, yearly, or range",
			)
	}

	filter := repositories.LeaveReportFilter{
		FromMonth: fromMonth,
		FromYear:  fromYear,

		ToMonth: toMonth,
		ToYear:  toYear,

		Search: req.Search,
		Role:   req.Role,

		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	records, err := s.repo.GetLeaveReportByRange(filter)
	if err != nil {
		return models.LeaveReportResponse{},
			fmt.Errorf("failed to fetch leave report: %w", err)
	}

	if records == nil {
		records = []models.LeaveReportRecord{}
	}

	return models.LeaveReportResponse{
		ReportType: req.ReportType,

		FromMonth: fromMonth,
		FromYear:  fromYear,

		ToMonth: toMonth,
		ToYear:  toYear,

		Total:   len(records),
		Records: records,
	}, nil
}
