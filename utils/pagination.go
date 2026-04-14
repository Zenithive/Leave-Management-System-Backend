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

// GetFilterParams extracts filtering and sorting parameters from query string
// All parameters are optional
func GetFilterParams(c *gin.Context) FilterParams {
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "name")  // Default sort by name
	sortDir := c.DefaultQuery("sort_dir", "asc") // Default ascending

	// Validate sort_by (allow common fields, specific validation in repository)
	validSortFields := map[string]bool{
		"name":               true,
		"created_at":         true,
		"category":           true,
		"price":              true,
		"total_quantity":     true,
		"remaining_quantity": true,
		"is_shared":          true,
		"purchase_date":      true,
	}
	if !validSortFields[sortBy] {
		sortBy = "name" // Fallback to name if invalid
	}

	// Validate sort_dir
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc" // Fallback to asc if invalid
	}

	return FilterParams{
		Search:  search,
		SortBy:  sortBy,
		SortDir: sortDir,
	}
}
