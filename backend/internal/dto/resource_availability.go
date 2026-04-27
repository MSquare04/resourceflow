package dto

import "time"

type CreateResourceAvailabilityRequest struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type UpdateResourceAvailabilityRequest struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type ResourceAvailabilityResponse struct {
	ID         int64     `json:"id"`
	ResourceID int64     `json:"resource_id"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
