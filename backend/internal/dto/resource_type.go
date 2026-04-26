package dto

type CreateResourceTypeRequest struct {
	CategoryID  int64  `json:"category_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateResourceTypeRequest struct {
	CategoryID  int64  `json:"category_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type ResourceTypeResponse struct {
	ID          int64  `json:"id"`
	CategoryID  int64  `json:"category_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}
