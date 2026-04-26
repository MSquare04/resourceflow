package model

type User struct {
	ID           int64
	FullName     string
	Email        string
	PasswordHash string
	DepartmentID *int64
	IsActive     bool
}
