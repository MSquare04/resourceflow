package dto

import "time"

type CreateBookingRequest struct {
	ResourceID int64     `json:"resource_id"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Purpose    *string   `json:"purpose"`
}

type BookingResponse struct {
	ID               int64      `json:"id"`
	ResourceID       int64      `json:"resource_id"`
	ResourceName     string     `json:"resource_name"`
	UserID           int64      `json:"user_id"`
	UserFullName     *string    `json:"user_full_name,omitempty"`
	StartAt          time.Time  `json:"start_at"`
	EndAt            time.Time  `json:"end_at"`
	Purpose          *string    `json:"purpose"`
	Status           string     `json:"status"`
	ApprovedByUserID *int64     `json:"approved_by_user_id"`
	ApprovedAt       *time.Time `json:"approved_at"`
	CancelledAt      *time.Time `json:"cancelled_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
