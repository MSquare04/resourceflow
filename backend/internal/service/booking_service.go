package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
)

var (
	ErrBookingNotFound            = errors.New("booking not found")
	ErrBookingConflict            = errors.New("booking conflict")
	ErrBookingOutOfAvailability   = errors.New("booking is outside resource availability")
	ErrBookingRuleNotConfigured   = errors.New("active booking rule not configured")
	ErrBookingLimitExceeded       = errors.New("max active bookings per user exceeded")
	ErrBookingHorizonExceeded     = errors.New("booking horizon exceeded")
	ErrBookingForbidden           = errors.New("booking action is forbidden")
	ErrBookingInvalidStatusAction = errors.New("invalid booking status transition")
	ErrBookingStartNotFuture      = errors.New("booking start time must be in the future")
	ErrBookingResourceUnavailable = errors.New("resource is inactive or not bookable")
	ErrBookingCompleteTooEarly    = errors.New("booking cannot be completed before end_at")
)

var activeBookingStatuses = []string{
	model.BookingStatusPending,
	model.BookingStatusConfirmed,
}

type BookingService struct {
	bookings     repository.BookingRepository
	resources    repository.ResourceRepository
	users        repository.UserRepository
	bookingRules repository.BookingRuleRepository
}

func NewBookingService(
	bookings repository.BookingRepository,
	resources repository.ResourceRepository,
	users repository.UserRepository,
	bookingRules repository.BookingRuleRepository,
) *BookingService {
	return &BookingService{
		bookings:     bookings,
		resources:    resources,
		users:        users,
		bookingRules: bookingRules,
	}
}

func (s *BookingService) Create(ctx context.Context, userID int64, req dto.CreateBookingRequest) (dto.BookingResponse, error) {
	if userID <= 0 || req.ResourceID <= 0 {
		logBookingRejection(ctx, "booking create rejected: invalid identifiers",
			"actor_user_id", userID,
			"resource_id", req.ResourceID,
		)
		return dto.BookingResponse{}, ErrValidation
	}

	startAt, endAt, err := validateBookingRange(req.StartAt, req.EndAt)
	if err != nil {
		logBookingRejection(ctx, "booking create rejected: invalid time range",
			"actor_user_id", userID,
			"resource_id", req.ResourceID,
			"start_at", req.StartAt,
			"end_at", req.EndAt,
		)
		return dto.BookingResponse{}, err
	}
	if !startAt.After(time.Now().UTC()) {
		logBookingRejection(ctx, "booking create rejected: start time is not in the future",
			"actor_user_id", userID,
			"resource_id", req.ResourceID,
			"start_at", startAt,
			"end_at", endAt,
		)
		return dto.BookingResponse{}, ErrBookingStartNotFuture
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.BookingResponse{}, ErrValidation
		}
		return dto.BookingResponse{}, err
	}
	if !user.IsActive {
		return dto.BookingResponse{}, ErrValidation
	}

	resource, err := s.resources.FindByID(ctx, req.ResourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.BookingResponse{}, ErrResourceNotFound
		}
		return dto.BookingResponse{}, err
	}
	if !resource.IsActive || !resource.IsBookable {
		logBookingRejection(ctx, "booking create rejected: resource is inactive or not bookable",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
		)
		return dto.BookingResponse{}, ErrBookingResourceUnavailable
	}

	covered, err := s.bookings.IsCoveredByAvailability(ctx, req.ResourceID, startAt, endAt)
	if err != nil {
		return dto.BookingResponse{}, err
	}
	if !covered {
		logBookingRejection(ctx, "booking create rejected: outside resource availability",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
		)
		return dto.BookingResponse{}, ErrBookingOutOfAvailability
	}

	rule, err := s.bookingRules.FindActiveByResourceTypeID(ctx, resource.TypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.BookingResponse{}, ErrBookingRuleNotConfigured
		}
		return dto.BookingResponse{}, err
	}

	durationMinutes := endAt.Sub(startAt).Minutes()
	if durationMinutes < float64(rule.MinDurationMinutes) || durationMinutes > float64(rule.MaxDurationMinutes) {
		logBookingRejection(ctx, "booking create rejected: duration violates booking rules",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
			"duration_minutes", durationMinutes,
			"rule_min_duration_minutes", rule.MinDurationMinutes,
			"rule_max_duration_minutes", rule.MaxDurationMinutes,
		)
		return dto.BookingResponse{}, ErrValidation
	}

	activeCount, err := s.bookings.CountByUserAndStatuses(ctx, userID, activeBookingStatuses)
	if err != nil {
		return dto.BookingResponse{}, err
	}
	if activeCount >= int64(rule.MaxActiveBookingsPerUser) {
		logBookingRejection(ctx, "booking create rejected: max active bookings per user exceeded",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"active_bookings_count", activeCount,
			"rule_max_active_bookings_per_user", rule.MaxActiveBookingsPerUser,
			"start_at", startAt,
			"end_at", endAt,
		)
		return dto.BookingResponse{}, ErrBookingLimitExceeded
	}

	horizonBoundary := time.Now().UTC().AddDate(0, 0, int(rule.BookingHorizonDays))
	if endAt.After(horizonBoundary) {
		logBookingRejection(ctx, "booking create rejected: booking horizon exceeded",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
			"horizon_boundary", horizonBoundary,
			"rule_booking_horizon_days", rule.BookingHorizonDays,
		)
		return dto.BookingResponse{}, ErrBookingHorizonExceeded
	}

	hasConflict, err := s.bookings.HasConflict(ctx, req.ResourceID, startAt, endAt, activeBookingStatuses)
	if err != nil {
		return dto.BookingResponse{}, err
	}
	if hasConflict {
		logBookingRejection(ctx, "booking create rejected: overlap with active booking",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
		)
		return dto.BookingResponse{}, ErrBookingConflict
	}

	status := model.BookingStatusConfirmed
	if rule.RequiresApproval {
		status = model.BookingStatusPending
	}

	booking, err := s.bookings.Create(ctx, repository.CreateBookingParams{
		ResourceID: req.ResourceID,
		UserID:     userID,
		StartAt:    startAt,
		EndAt:      endAt,
		Purpose:    trimNullableString(req.Purpose),
		Status:     status,
	})
	if err != nil {
		if isExclusionViolation(err) {
			logBookingRejection(ctx, "booking create rejected: overlap detected by database constraint",
				"resource_id", req.ResourceID,
				"actor_user_id", userID,
				"start_at", startAt,
				"end_at", endAt,
			)
			return dto.BookingResponse{}, ErrBookingConflict
		}
		if isForeignKeyViolation(err) || isCheckViolation(err) {
			return dto.BookingResponse{}, ErrValidation
		}
		return dto.BookingResponse{}, err
	}

	logBookingEvent(ctx, "booking created",
		"booking_id", booking.ID,
		"resource_id", booking.ResourceID,
		"booking_user_id", booking.UserID,
		"actor_user_id", userID,
		"status", booking.Status,
		"status_from", "",
		"status_to", booking.Status,
		"start_at", booking.StartAt.UTC(),
		"end_at", booking.EndAt.UTC(),
	)

	return mapBookingResponse(booking), nil
}

func (s *BookingService) List(ctx context.Context) ([]dto.BookingResponse, error) {
	bookings, err := s.bookings.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := mapBookingResponses(bookings)
	if err := s.enrichBookingResponses(ctx, responses, true); err != nil {
		return nil, err
	}

	return responses, nil
}

func (s *BookingService) ListByUserID(ctx context.Context, userID int64) ([]dto.BookingResponse, error) {
	if userID <= 0 {
		return nil, ErrValidation
	}

	bookings, err := s.bookings.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := mapBookingResponses(bookings)
	if err := s.enrichBookingResponses(ctx, responses, false); err != nil {
		return nil, err
	}

	return responses, nil
}

func (s *BookingService) ListBusyIntervalsByResourceID(ctx context.Context, resourceID int64) ([]dto.ResourceBusyIntervalResponse, error) {
	if resourceID <= 0 {
		return nil, ErrValidation
	}

	if _, err := s.resources.FindByID(ctx, resourceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	until := now.AddDate(0, 0, 30)
	bookings, err := s.bookings.ListBusyIntervalsByResourceID(ctx, resourceID, activeBookingStatuses, now, until)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ResourceBusyIntervalResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, dto.ResourceBusyIntervalResponse{
			StartAt: booking.StartAt.UTC(),
			EndAt:   booking.EndAt.UTC(),
		})
	}

	return result, nil
}

func (s *BookingService) GetByID(ctx context.Context, id int64, includeUserFullName bool) (dto.BookingResponse, error) {
	if id <= 0 {
		return dto.BookingResponse{}, ErrValidation
	}

	booking, err := s.bookings.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.BookingResponse{}, ErrBookingNotFound
		}
		return dto.BookingResponse{}, err
	}

	response := mapBookingResponse(booking)
	responses := []dto.BookingResponse{response}
	if err := s.enrichBookingResponses(ctx, responses, includeUserFullName); err != nil {
		return dto.BookingResponse{}, err
	}
	response = responses[0]

	return response, nil
}

func (s *BookingService) Cancel(ctx context.Context, id int64, actorUserID int64, isPrivileged bool) (dto.BookingResponse, error) {
	booking, err := s.getBookingForAction(ctx, id)
	if err != nil {
		return dto.BookingResponse{}, err
	}

	if !isPrivileged && booking.UserID != actorUserID {
		return dto.BookingResponse{}, ErrBookingNotFound
	}

	if booking.Status != model.BookingStatusPending && booking.Status != model.BookingStatusConfirmed {
		logBookingRejection(ctx, "booking cancel rejected: invalid status transition",
			"booking_id", booking.ID,
			"resource_id", booking.ResourceID,
			"booking_user_id", booking.UserID,
			"actor_user_id", actorUserID,
			"status", booking.Status,
			"status_from", booking.Status,
			"status_to", model.BookingStatusCancelled,
			"start_at", booking.StartAt.UTC(),
			"end_at", booking.EndAt.UTC(),
		)
		return dto.BookingResponse{}, ErrBookingInvalidStatusAction
	}

	now := time.Now().UTC()
	updated, statusFrom, err := s.bookings.TransitionStatus(ctx, id, []string{
		model.BookingStatusPending,
		model.BookingStatusConfirmed,
	}, repository.UpdateBookingStatusParams{
		Status:           model.BookingStatusCancelled,
		ApprovedByUserID: nil,
		ApprovedAt:       nil,
		CancelledAt:      &now,
		CompletedAt:      nil,
	})
	if err != nil {
		return dto.BookingResponse{}, s.handleBookingTransitionFailure(
			ctx,
			id,
			actorUserID,
			model.BookingStatusCancelled,
			err,
		)
	}

	logBookingEvent(ctx, "booking cancelled",
		"booking_id", updated.ID,
		"resource_id", updated.ResourceID,
		"booking_user_id", updated.UserID,
		"actor_user_id", actorUserID,
		"status", updated.Status,
		"status_from", statusFrom,
		"status_to", updated.Status,
		"start_at", updated.StartAt.UTC(),
		"end_at", updated.EndAt.UTC(),
	)

	return mapBookingResponse(updated), nil
}

func (s *BookingService) Complete(ctx context.Context, id int64, actorUserID int64, isPrivileged bool) (dto.BookingResponse, error) {
	return s.CompleteAt(ctx, id, actorUserID, isPrivileged, time.Now().UTC())
}

func (s *BookingService) CompleteAt(
	ctx context.Context,
	id int64,
	actorUserID int64,
	isPrivileged bool,
	now time.Time,
) (dto.BookingResponse, error) {
	booking, err := s.getBookingForAction(ctx, id)
	if err != nil {
		return dto.BookingResponse{}, err
	}

	if !isPrivileged && booking.UserID != actorUserID {
		return dto.BookingResponse{}, ErrBookingNotFound
	}

	if booking.Status != model.BookingStatusConfirmed {
		logBookingRejection(ctx, "booking complete rejected: invalid status transition",
			"booking_id", booking.ID,
			"resource_id", booking.ResourceID,
			"booking_user_id", booking.UserID,
			"actor_user_id", actorUserID,
			"status", booking.Status,
			"status_from", booking.Status,
			"status_to", model.BookingStatusCompleted,
			"start_at", booking.StartAt.UTC(),
			"end_at", booking.EndAt.UTC(),
		)
		return dto.BookingResponse{}, ErrBookingInvalidStatusAction
	}

	now = now.UTC()
	if booking.EndAt.UTC().After(now) {
		logBookingRejection(ctx, "booking complete rejected: booking has not ended yet",
			"booking_id", booking.ID,
			"resource_id", booking.ResourceID,
			"booking_user_id", booking.UserID,
			"actor_user_id", actorUserID,
			"status", booking.Status,
			"status_from", booking.Status,
			"status_to", model.BookingStatusCompleted,
			"start_at", booking.StartAt.UTC(),
			"end_at", booking.EndAt.UTC(),
			"now", now,
		)
		return dto.BookingResponse{}, ErrBookingCompleteTooEarly
	}

	updated, statusFrom, err := s.bookings.TransitionStatus(ctx, id, []string{
		model.BookingStatusConfirmed,
	}, repository.UpdateBookingStatusParams{
		Status:           model.BookingStatusCompleted,
		ApprovedByUserID: nil,
		ApprovedAt:       nil,
		CancelledAt:      booking.CancelledAt,
		CompletedAt:      &now,
	})
	if err != nil {
		return dto.BookingResponse{}, s.handleBookingTransitionFailure(
			ctx,
			id,
			actorUserID,
			model.BookingStatusCompleted,
			err,
		)
	}

	logBookingEvent(ctx, "booking completed",
		"booking_id", updated.ID,
		"resource_id", updated.ResourceID,
		"booking_user_id", updated.UserID,
		"actor_user_id", actorUserID,
		"status", updated.Status,
		"status_from", statusFrom,
		"status_to", updated.Status,
		"start_at", updated.StartAt.UTC(),
		"end_at", updated.EndAt.UTC(),
	)

	return mapBookingResponse(updated), nil
}

func (s *BookingService) ProcessExpiredBookings(
	ctx context.Context,
	now time.Time,
) (repository.ExpiredBookingProcessingResult, error) {
	now = now.UTC()
	result, err := s.bookings.ProcessExpired(ctx, now)
	if err != nil {
		return repository.ExpiredBookingProcessingResult{}, err
	}

	if result.CompletedCount > 0 || result.CancelledCount > 0 {
		logBookingEvent(ctx, "expired bookings processed",
			"completed_count", result.CompletedCount,
			"cancelled_count", result.CancelledCount,
			"processed_at", now,
		)
	}

	return result, nil
}

func (s *BookingService) Approve(ctx context.Context, id int64, approverUserID int64) (dto.BookingResponse, error) {
	now := time.Now().UTC()
	updated, statusFrom, err := s.bookings.TransitionStatus(ctx, id, []string{
		model.BookingStatusPending,
	}, repository.UpdateBookingStatusParams{
		Status:           model.BookingStatusConfirmed,
		ApprovedByUserID: &approverUserID,
		ApprovedAt:       &now,
		CancelledAt:      nil,
		CompletedAt:      nil,
	})
	if err != nil {
		return dto.BookingResponse{}, s.handleBookingTransitionFailure(
			ctx,
			id,
			approverUserID,
			model.BookingStatusConfirmed,
			err,
		)
	}

	logBookingEvent(ctx, "booking approved",
		"booking_id", updated.ID,
		"resource_id", updated.ResourceID,
		"booking_user_id", updated.UserID,
		"actor_user_id", approverUserID,
		"status", updated.Status,
		"status_from", statusFrom,
		"status_to", updated.Status,
		"start_at", updated.StartAt.UTC(),
		"end_at", updated.EndAt.UTC(),
	)

	return mapBookingResponse(updated), nil
}

func (s *BookingService) Reject(ctx context.Context, id int64, approverUserID int64) (dto.BookingResponse, error) {
	now := time.Now().UTC()
	updated, statusFrom, err := s.bookings.TransitionStatus(ctx, id, []string{
		model.BookingStatusPending,
	}, repository.UpdateBookingStatusParams{
		Status:           model.BookingStatusRejected,
		ApprovedByUserID: &approverUserID,
		ApprovedAt:       &now,
		CancelledAt:      nil,
		CompletedAt:      nil,
	})
	if err != nil {
		return dto.BookingResponse{}, s.handleBookingTransitionFailure(
			ctx,
			id,
			approverUserID,
			model.BookingStatusRejected,
			err,
		)
	}

	logBookingEvent(ctx, "booking rejected",
		"booking_id", updated.ID,
		"resource_id", updated.ResourceID,
		"booking_user_id", updated.UserID,
		"actor_user_id", approverUserID,
		"status", updated.Status,
		"status_from", statusFrom,
		"status_to", updated.Status,
		"start_at", updated.StartAt.UTC(),
		"end_at", updated.EndAt.UTC(),
	)

	return mapBookingResponse(updated), nil
}

func (s *BookingService) getBookingForAction(ctx context.Context, id int64) (model.Booking, error) {
	if id <= 0 {
		return model.Booking{}, ErrValidation
	}

	booking, err := s.bookings.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Booking{}, ErrBookingNotFound
		}
		return model.Booking{}, err
	}

	return booking, nil
}

func normalizeBookingUpdateError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrBookingNotFound
	}
	if isExclusionViolation(err) {
		return ErrBookingConflict
	}
	if isForeignKeyViolation(err) || isCheckViolation(err) {
		return ErrValidation
	}
	return err
}

func (s *BookingService) handleBookingTransitionFailure(
	ctx context.Context,
	bookingID int64,
	actorUserID int64,
	statusTo string,
	err error,
) error {
	if !errors.Is(err, sql.ErrNoRows) {
		return normalizeBookingUpdateError(err)
	}

	current, findErr := s.bookings.FindByID(ctx, bookingID)
	if findErr != nil {
		if errors.Is(findErr, sql.ErrNoRows) {
			return ErrBookingNotFound
		}
		return findErr
	}

	logBookingRejection(ctx, "booking action rejected: invalid status transition",
		"booking_id", current.ID,
		"resource_id", current.ResourceID,
		"booking_user_id", current.UserID,
		"actor_user_id", actorUserID,
		"status", current.Status,
		"status_from", current.Status,
		"status_to", statusTo,
		"start_at", current.StartAt.UTC(),
		"end_at", current.EndAt.UTC(),
	)

	return ErrBookingInvalidStatusAction
}

func validateBookingRange(startAt, endAt time.Time) (time.Time, time.Time, error) {
	if startAt.IsZero() || endAt.IsZero() || !startAt.Before(endAt) {
		return time.Time{}, time.Time{}, ErrValidation
	}
	return startAt.UTC(), endAt.UTC(), nil
}

func mapBookingResponses(bookings []model.Booking) []dto.BookingResponse {
	result := make([]dto.BookingResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, mapBookingResponse(booking))
	}
	return result
}

func mapBookingResponse(booking model.Booking) dto.BookingResponse {
	purpose := booking.Purpose
	if purpose != nil {
		trimmed := strings.TrimSpace(*purpose)
		if trimmed == "" {
			purpose = nil
		} else {
			purpose = &trimmed
		}
	}

	return dto.BookingResponse{
		ID:               booking.ID,
		ResourceID:       booking.ResourceID,
		UserID:           booking.UserID,
		StartAt:          booking.StartAt.UTC(),
		EndAt:            booking.EndAt.UTC(),
		Purpose:          purpose,
		Status:           booking.Status,
		ApprovedByUserID: booking.ApprovedByUserID,
		ApprovedAt:       nullableUTCTime(booking.ApprovedAt),
		CancelledAt:      nullableUTCTime(booking.CancelledAt),
		CompletedAt:      nullableUTCTime(booking.CompletedAt),
		CreatedAt:        booking.CreatedAt.UTC(),
		UpdatedAt:        booking.UpdatedAt.UTC(),
	}
}

func (s *BookingService) enrichBookingResponses(ctx context.Context, responses []dto.BookingResponse, includeUserFullName bool) error {
	resourceNames := make(map[int64]string, len(responses))
	userNames := make(map[int64]string, len(responses))

	for i := range responses {
		resourceName, ok := resourceNames[responses[i].ResourceID]
		if !ok {
			resource, err := s.resources.FindByID(ctx, responses[i].ResourceID)
			if err != nil {
				return err
			}
			resourceName = resource.Name
			resourceNames[responses[i].ResourceID] = resourceName
		}
		responses[i].ResourceName = resourceName

		if includeUserFullName {
			userFullName, ok := userNames[responses[i].UserID]
			if !ok {
				user, err := s.users.FindByID(ctx, responses[i].UserID)
				if err != nil {
					return err
				}
				userFullName = user.FullName
				userNames[responses[i].UserID] = userFullName
			}
			nameCopy := userFullName
			responses[i].UserFullName = &nameCopy
		}
	}

	return nil
}

func nullableUTCTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func logBookingEvent(ctx context.Context, message string, attrs ...any) {
	args := append([]any{"request_id", requestIDFromContext(ctx)}, attrs...)
	slog.Default().Info(message, args...)
}

func logBookingRejection(ctx context.Context, message string, attrs ...any) {
	args := append([]any{"request_id", requestIDFromContext(ctx)}, attrs...)
	slog.Default().Warn(message, args...)
}
