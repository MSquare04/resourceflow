package model

type ResourceType struct {
	ID          int64
	CategoryID  int64
	Code        string
	Name        string
	Description string
	IsActive    bool
}
