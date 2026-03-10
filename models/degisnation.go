package models

// ----------------- DESIGNATION-----------------
type Designation struct {
	ID              *string `json:"id" db:"id"`
	DesignationName string  `json:"designation_name" db:"designation_name"`
	Description     *string `json:"description,omitempty" db:"description"`
}

type DesignationInput struct {
	DesignationName string  `json:"designation_name" validate:"required"`
	Description     *string `json:"description,omitempty"`
}
