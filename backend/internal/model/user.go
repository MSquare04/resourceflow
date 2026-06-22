package model

type User struct {
	ID           int64
	FullName     string
	Email        string
	PasswordHash string
	AuthVersion  int
	DepartmentID *int64
	IsActive     bool
}
