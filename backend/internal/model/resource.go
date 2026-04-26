package model

type Resource struct {
	ID           int64
	Name         string
	Description  string
	CategoryID   int64
	TypeID       int64
	DepartmentID *int64
	Location     *string
	Capacity     *int64
	IsBookable   bool
	IsActive     bool
}
