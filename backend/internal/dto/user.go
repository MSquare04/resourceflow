package dto

type CreateUserRequest struct {
	FullName     string   `json:"full_name"`
	Email        string   `json:"email"`
	Password     string   `json:"password"`
	DepartmentID *int64   `json:"department_id"`
	IsActive     *bool    `json:"is_active"`
	Roles        []string `json:"roles"`
}

type UpdateUserRequest struct {
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	DepartmentID *int64 `json:"department_id"`
	IsActive     *bool  `json:"is_active"`
}

type UpdateUserRolesRequest struct {
	Roles []string `json:"roles"`
}

type UserResponse struct {
	ID           int64    `json:"id"`
	FullName     string   `json:"full_name"`
	Email        string   `json:"email"`
	DepartmentID *int64   `json:"department_id"`
	IsActive     bool     `json:"is_active"`
	Roles        []string `json:"roles"`
}
