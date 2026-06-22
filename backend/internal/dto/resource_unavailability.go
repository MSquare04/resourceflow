package dto

import "time"

type CreateResourceUnavailabilityRequest struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Reason  *string   `json:"reason"`
}

type UpdateResourceUnavailabilityRequest struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Reason  *string   `json:"reason"`
}

type ResourceUnavailabilityResponse struct {
	ID         int64     `json:"id"`
	ResourceID int64     `json:"resource_id"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Reason     *string   `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
