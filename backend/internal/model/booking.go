package model

import "time"

const (
	BookingStatusPending   = "pending"
	BookingStatusConfirmed = "confirmed"
	BookingStatusRejected  = "rejected"
	BookingStatusCancelled = "cancelled"
	BookingStatusCompleted = "completed"
)

type Booking struct {
	ID               int64
	ResourceID       int64
	UserID           int64
	StartAt          time.Time
	EndAt            time.Time
	Purpose          *string
	Status           string
	ApprovedByUserID *int64
	ApprovedAt       *time.Time
	CancelledAt      *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
