package dto

import "time"

type CreateBookingRuleRequest struct {
	ResourceTypeID           int64 `json:"resource_type_id"`
	MinDurationMinutes       int32 `json:"min_duration_minutes"`
	MaxDurationMinutes       int32 `json:"max_duration_minutes"`
	MaxActiveBookingsPerUser int32 `json:"max_active_bookings_per_user"`
	RequiresApproval         *bool `json:"requires_approval"`
	BookingHorizonDays       int32 `json:"booking_horizon_days"`
	IsActive                 *bool `json:"is_active"`
}

type UpdateBookingRuleRequest struct {
	ResourceTypeID           int64 `json:"resource_type_id"`
	MinDurationMinutes       int32 `json:"min_duration_minutes"`
	MaxDurationMinutes       int32 `json:"max_duration_minutes"`
	MaxActiveBookingsPerUser int32 `json:"max_active_bookings_per_user"`
	RequiresApproval         *bool `json:"requires_approval"`
	BookingHorizonDays       int32 `json:"booking_horizon_days"`
	IsActive                 *bool `json:"is_active"`
}

type BookingRuleResponse struct {
	ID                       int64     `json:"id"`
	ResourceTypeID           int64     `json:"resource_type_id"`
	MinDurationMinutes       int32     `json:"min_duration_minutes"`
	MaxDurationMinutes       int32     `json:"max_duration_minutes"`
	MaxActiveBookingsPerUser int32     `json:"max_active_bookings_per_user"`
	RequiresApproval         bool      `json:"requires_approval"`
	BookingHorizonDays       int32     `json:"booking_horizon_days"`
	IsActive                 bool      `json:"is_active"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
