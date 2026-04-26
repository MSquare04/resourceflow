package dto

type CreateResourceRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	CategoryID   int64   `json:"category_id"`
	TypeID       int64   `json:"type_id"`
	DepartmentID *int64  `json:"department_id"`
	Location     *string `json:"location"`
	Capacity     *int64  `json:"capacity"`
	IsBookable   *bool   `json:"is_bookable"`
	IsActive     *bool   `json:"is_active"`
}

type UpdateResourceRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	CategoryID   int64   `json:"category_id"`
	TypeID       int64   `json:"type_id"`
	DepartmentID *int64  `json:"department_id"`
	Location     *string `json:"location"`
	Capacity     *int64  `json:"capacity"`
	IsBookable   *bool   `json:"is_bookable"`
	IsActive     *bool   `json:"is_active"`
}

type ResourceResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	CategoryID   int64   `json:"category_id"`
	TypeID       int64   `json:"type_id"`
	DepartmentID *int64  `json:"department_id"`
	Location     *string `json:"location"`
	Capacity     *int64  `json:"capacity"`
	IsBookable   bool    `json:"is_bookable"`
	IsActive     bool    `json:"is_active"`
}
