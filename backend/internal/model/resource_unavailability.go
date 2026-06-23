package model

import "time"

type ResourceUnavailability struct {
	ID         int64
	ResourceID int64
	StartAt    time.Time
	EndAt      time.Time
	Reason     *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
