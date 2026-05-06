package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"-"`
}

// FilterParams holds filtering and sorting parameters
type FilterParams struct {
	Search  string // Search term for name
	SortBy  string // Field to sort by (name, created_at)
	SortDir string // Sort direction (asc, desc)
}

// PaginationResponse holds pagination metadata
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// GetPaginationParams extracts pagination parameters from query string
// If no pagination params provided, returns PageSize=0 (meaning no pagination)
// Default when provided: page=1, page_size=10
// Max page_size: 100
func GetPaginationParams(c *gin.Context) PaginationParams {
	// Check if pagination params exist
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	// If neither param is provided, return no pagination
	if pageStr == "" && pageSizeStr == "" {
		return PaginationParams{
			Page:     0,
			PageSize: 0,
			Offset:   0,
		}
	}

	// Parse pagination params with defaults
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Validate page
	if page < 1 {
		page = 1
	}

	// Validate page_size
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
	}
}

// CalculatePaginationResponse creates pagination metadata
func CalculatePaginationResponse(page, pageSize int, totalItems int64) PaginationResponse {
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}

	return PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

// GetFilterParams extracts filtering and sorting parameters from query string.
// validSortFields is a set of allowed sort_by values for the calling entity.
// Pass nil to skip sort_by validation (not recommended).
func GetFilterParams(c *gin.Context, validSortFields map[string]bool) FilterParams {
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "name")
	sortDir := c.DefaultQuery("sort_dir", "asc")

	// Validate sort_by against caller-provided allowed fields
	if validSortFields != nil && !validSortFields[sortBy] {
		sortBy = "name"
	}

	// Validate sort_dir
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	return FilterParams{
		Search:  search,
		SortBy:  sortBy,
		SortDir: sortDir,
	}
}

// CategorySortFields defines valid sort fields for equipment categories
var CategorySortFields = map[string]bool{
	"name":       true,
	"created_at": true,
}

// EquipmentSortFields defines valid sort fields for equipment
var EquipmentSortFields = map[string]bool{
	"name":               true,
	"category":           true,
	"price":              true,
	"total_quantity":     true,
	"remaining_quantity": true,
	"is_shared":          true,
	"purchase_date":      true,
	"created_at":         true,
}

// AssignmentSortFields defines valid sort fields for equipment assignments
var AssignmentSortFields = map[string]bool{
	"employee_name":  true,
	"equipment_name": true,
	"quantity":       true,
	"assigned_at":    true,
}
