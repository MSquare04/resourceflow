package model

import "time"

type ResourceAvailability struct {
	ID         int64
	ResourceID int64
	StartAt    time.Time
	EndAt      time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
