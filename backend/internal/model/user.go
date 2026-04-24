package model

type User struct {
	ID           int64
	FullName     string
	Email        string
	PasswordHash string
	IsActive     bool
}
