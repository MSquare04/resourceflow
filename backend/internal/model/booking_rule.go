package model

import "time"

type BookingRule struct {
	ID                       int64
	ResourceTypeID           int64
	MinDurationMinutes       int32
	MaxDurationMinutes       int32
	MaxActiveBookingsPerUser int32
	RequiresApproval         bool
	BookingHorizonDays       int32
	IsActive                 bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
