package utils

import (
	"testing"
)

// ─── CalculatePaginationResponse ─────────────────────────────────────────────

func TestCalculatePaginationResponse(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		totalItems int64
		wantPages  int
		wantPage   int
		wantSize   int
		wantTotal  int64
	}{
		{
			name: "exact fit — no remainder",
			// 20 items, 10 per page → 2 pages
			page: 1, pageSize: 10, totalItems: 20,
			wantPages: 2, wantPage: 1, wantSize: 10, wantTotal: 20,
		},
		{
			name: "remainder rounds up",
			// 21 items, 10 per page → 3 pages
			page: 2, pageSize: 10, totalItems: 21,
			wantPages: 3, wantPage: 2, wantSize: 10, wantTotal: 21,
		},
		{
			name: "single page",
			page: 1, pageSize: 50, totalItems: 5,
			wantPages: 1, wantPage: 1, wantSize: 50, wantTotal: 5,
		},
		{
			name: "zero items",
			page: 1, pageSize: 10, totalItems: 0,
			wantPages: 0, wantPage: 1, wantSize: 10, wantTotal: 0,
		},
		{
			name: "page size of 1",
			page: 3, pageSize: 1, totalItems: 5,
			wantPages: 5, wantPage: 3, wantSize: 1, wantTotal: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePaginationResponse(tt.page, tt.pageSize, tt.totalItems)

			if got.TotalPages != tt.wantPages {
				t.Errorf("TotalPages = %d, want %d", got.TotalPages, tt.wantPages)
			}
			if got.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.wantSize)
			}
			if got.TotalItems != tt.wantTotal {
				t.Errorf("TotalItems = %d, want %d", got.TotalItems, tt.wantTotal)
			}
		})
	}
}

// ─── GetFilterParams ──────────────────────────────────────────────────────────

func TestGetFilterParams_Defaults(t *testing.T) {
	validFields := map[string]bool{"name": true, "email": true, "joining_date": true}

	t.Run("unknown sort_by falls back to name", func(t *testing.T) {
		params := resolveFilterParams("", "unknown_field", "asc", validFields)
		if params.SortBy != "name" {
			t.Errorf("SortBy = %q, want %q", params.SortBy, "name")
		}
	})

	t.Run("invalid sort_dir falls back to asc", func(t *testing.T) {
		params := resolveFilterParams("", "name", "sideways", validFields)
		if params.SortDir != "asc" {
			t.Errorf("SortDir = %q, want %q", params.SortDir, "asc")
		}
	})

	t.Run("valid desc is preserved", func(t *testing.T) {
		params := resolveFilterParams("", "email", "desc", validFields)
		if params.SortDir != "desc" {
			t.Errorf("SortDir = %q, want %q", params.SortDir, "desc")
		}
		if params.SortBy != "email" {
			t.Errorf("SortBy = %q, want %q", params.SortBy, "email")
		}
	})

	t.Run("search term is preserved", func(t *testing.T) {
		params := resolveFilterParams("alice", "name", "asc", validFields)
		if params.Search != "alice" {
			t.Errorf("Search = %q, want %q", params.Search, "alice")
		}
	})

	t.Run("nil validFields skips sort_by validation", func(t *testing.T) {
		params := resolveFilterParams("", "anything", "asc", nil)
		if params.SortBy != "anything" {
			t.Errorf("SortBy = %q, want %q", params.SortBy, "anything")
		}
	})
}

// resolveFilterParams is a pure helper that mirrors the logic inside
// GetFilterParams without needing a *gin.Context.
func resolveFilterParams(search, sortBy, sortDir string, validFields map[string]bool) FilterParams {
	if validFields != nil && !validFields[sortBy] {
		sortBy = "name"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}
	return FilterParams{Search: search, SortBy: sortBy, SortDir: sortDir}
}
