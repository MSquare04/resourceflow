package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
)

var (
	ErrBookingNotFound            = errors.New("booking not found")
	ErrBookingConflict            = errors.New("booking conflict")
	ErrBookingRuleNotConfigured   = errors.New("active booking rule not configured")
	ErrBookingLimitExceeded       = errors.New("max active bookings per user exceeded")
	ErrBookingHorizonExceeded     = errors.New("booking horizon exceeded")
	ErrBookingOutsideWorkday      = errors.New("booking interval is outside booking rule workday")
	ErrBookingInUnavailability    = errors.New("booking interval intersects resource unavailability")
	ErrBookingForbidden           = errors.New("booking action is forbidden")
	ErrBookingInvalidStatusAction = errors.New("invalid booking status transition")
	ErrBookingStartNotFuture      = errors.New("booking start time cannot be earlier than the current minute")
	ErrBookingResourceUnavailable = errors.New("resource is inactive or not bookable")
	ErrBookingCompleteTooEarly    = errors.New("booking cannot be completed before end_at")
	ErrBookingAlreadyEnded        = errors.New("booking has already ended")
	ErrBusyIntervalRangeInvalid   = errors.New("busy interval range is invalid")
	ErrBookingBatchTooLarge       = errors.New("booking batch exceeds the maximum allowed dates")
	ErrBookingBatchDuplicateDate  = errors.New("booking batch contains duplicate dates")
	ErrBookingBatchInvalid        = errors.New("booking batch contains invalid dates")
)

var activeBookingStatuses = []string{
	model.BookingStatusPending,
	model.BookingStatusConfirmed,
}

const maxBatchBookingDates = 31

const (
	batchErrorCodeStartInPast         = dto.ErrorCodeBookingStartInPast
	batchErrorCodeOutsideWorkday      = dto.ErrorCodeBookingOutsideWorkday
	batchErrorCodeInUnavailability    = dto.ErrorCodeBookingInUnavailability
	batchErrorCodeConflict            = dto.ErrorCodeBookingConflict
	batchErrorCodeResourceUnavailable = dto.ErrorCodeBookingResourceUnavailable
	batchErrorCodeRuleNotConfigured   = dto.ErrorCodeBookingRuleNotConfigured
	batchErrorCodeLimitExceeded       = dto.ErrorCodeBookingLimitExceeded
	batchErrorCodeHorizonExceeded     = dto.ErrorCodeBookingHorizonExceeded
)

type BookingService struct {
	bookings       repository.BookingRepository
	resources      repository.ResourceRepository
	users          repository.UserRepository
	bookingRules   repository.BookingRuleRepository
	appLocation    *time.Location
	unavailability bookingUnavailabilityChecker
}

type bookingUnavailabilityChecker interface {
	HasConflict(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error)
}

type noopBookingUnavailabilityChecker struct{}

func (noopBookingUnavailabilityChecker) HasConflict(context.Context, int64, time.Time, time.Time) (bool, error) {
	return false, nil
}

func NewBookingService(
	bookings repository.BookingRepository,
	resources repository.ResourceRepository,
	users repository.UserRepository,
	bookingRules repository.BookingRuleRepository,
) *BookingService {
	return &BookingService{
		bookings:       bookings,
		resources:      resources,
		users:          users,
		bookingRules:   bookingRules,
		appLocation:    time.UTC,
		unavailability: noopBookingUnavailabilityChecker{},
	}
}

func (s *BookingService) WithTimeLocation(location *time.Location) *BookingService {
	if location != nil {
		s.appLocation = location
	}
	return s
}

func (s *BookingService) WithUnavailabilityChecker(checker bookingUnavailabilityChecker) *BookingService {
	if checker != nil {
		s.unavailability = checker
	}
	return s
}

func (s *BookingService) Create(ctx context.Context, userID int64, req dto.CreateBookingRequest) (dto.BookingResponse, error) {
	return s.CreateAt(ctx, userID, req, time.Now().UTC())
}

func (s *BookingService) PreviewBatch(ctx context.Context, userID int64, req dto.BatchBookingRequest) (dto.BatchBookingPreviewResponse, error) {
	return s.PreviewBatchAt(ctx, userID, req, time.Now().UTC())
}

func (s *BookingService) PreviewBatchAt(
	ctx context.Context,
	userID int64,
	req dto.BatchBookingRequest,
	now time.Time,
) (dto.BatchBookingPreviewResponse, error) {
	prepared, err := s.prepareBatchBase(ctx, userID, req)
	if err != nil {
		return dto.BatchBookingPreviewResponse{}, err
	}

	items := s.evaluateBatchCandidates(ctx, s.bookings, prepared, now.UTC())
	return dto.BatchBookingPreviewResponse{
		CanCreate: allBatchItemsValid(items),
		Items:     items,
	}, nil
}

func (s *BookingService) CreateBatch(ctx context.Context, userID int64, req dto.BatchBookingRequest) (dto.BatchBookingCreateResponse, error) {
	return s.CreateBatchAt(ctx, userID, req, time.Now().UTC())
}

func (s *BookingService) CreateBatchAt(
	ctx context.Context,
	userID int64,
	req dto.BatchBookingRequest,
	now time.Time,
) (dto.BatchBookingCreateResponse, error) {
	prepared, err := s.prepareBatchBase(ctx, userID, req)
	if err != nil {
		return dto.BatchBookingCreateResponse{}, err
	}

	txRepo, ok := s.bookings.(repository.BookingRepositoryTxRunner)
	if !ok {
		return dto.BatchBookingCreateResponse{}, fmt.Errorf("booking repository does not support transactions")
	}

	var created []model.Booking
	err = txRepo.WithTransaction(ctx, func(repo repository.BookingRepository) error {
		activeCount, countErr := repo.CountByUserAndStatuses(ctx, userID, activeBookingStatuses)
		if countErr != nil {
			return countErr
		}
		prepared.ActiveCount = activeCount

		items := s.evaluateBatchCandidates(ctx, repo, prepared, now.UTC())
		if !allBatchItemsValid(items) {
			return &BookingBatchValidationError{Items: items}
		}

		for _, candidate := range prepared.Candidates {
			booking, createErr := repo.Create(ctx, repository.CreateBookingParams{
				ResourceID: prepared.Resource.ID,
				UserID:     userID,
				StartAt:    candidate.StartAt,
				EndAt:      candidate.EndAt,
				Purpose:    trimNullableString(req.Purpose),
				Status:     prepared.Status,
			})
			if createErr != nil {
				if isExclusionViolation(createErr) {
					return &BookingBatchValidationError{
						Items: []dto.BatchBookingPreviewItem{{
							Date:      candidate.Date,
							Valid:     false,
							ErrorCode: stringPtr(batchErrorCodeConflict),
						}},
					}
				}
				if isForeignKeyViolation(createErr) || isCheckViolation(createErr) {
					return ErrValidation
				}
				return createErr
			}
			created = append(created, booking)
		}

		return nil
	})
	if err != nil {
		return dto.BatchBookingCreateResponse{}, err
	}

	items := mapBookingResponses(created)
	if err := s.enrichBookingResponses(ctx, items, false); err != nil {
		return dto.BatchBookingCreateResponse{}, err
	}

	return dto.BatchBookingCreateResponse{
		CreatedCount: len(items),
		Items:        items,
	}, nil
}

func (s *BookingService) CreateAt(
	ctx context.Context,
	userID int64,
	req dto.CreateBookingRequest,
	now time.Time,
) (dto.BookingResponse, error) {
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
	now = now.UTC()
	currentMinute := now.Truncate(time.Minute)
	if startAt.Before(currentMinute) {
		logBookingRejection(ctx, "booking create rejected: start time is not in the future",
			"actor_user_id", userID,
			"resource_id", req.ResourceID,
			"start_at", startAt,
			"end_at", endAt,
			"current_minute", currentMinute,
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

	horizonBoundary := now.AddDate(0, 0, int(rule.BookingHorizonDays))
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

	if err := s.ensureWithinRuleWorkday(startAt, endAt, rule); err != nil {
		logBookingRejection(ctx, "booking create rejected: outside booking rule workday",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
			"rule_workday_start", rule.WorkdayStart,
			"rule_workday_end", rule.WorkdayEnd,
			"rule_unrestricted_time", rule.UnrestrictedTime,
		)
		return dto.BookingResponse{}, err
	}

	hasUnavailabilityConflict, err := s.unavailability.HasConflict(ctx, req.ResourceID, startAt, endAt)
	if err != nil {
		return dto.BookingResponse{}, err
	}
	if hasUnavailabilityConflict {
		logBookingRejection(ctx, "booking create rejected: overlap with resource unavailability",
			"resource_id", req.ResourceID,
			"actor_user_id", userID,
			"start_at", startAt,
			"end_at", endAt,
		)
		return dto.BookingResponse{}, ErrBookingInUnavailability
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

type bookingBatchCandidate struct {
	Date    string
	StartAt time.Time
	EndAt   time.Time
}

type preparedBatchBooking struct {
	Resource   model.Resource
	Rule       model.BookingRule
	Status     string
	ActiveCount int64
	Candidates []bookingBatchCandidate
}

type BookingBatchValidationError struct {
	Items []dto.BatchBookingPreviewItem
}

func (e *BookingBatchValidationError) Error() string {
	return "booking batch contains invalid dates"
}

func (e *BookingBatchValidationError) FirstErrorCode() string {
	for _, item := range e.Items {
		if item.ErrorCode != nil && *item.ErrorCode != "" {
			return *item.ErrorCode
		}
	}
	return dto.ErrorCodeBookingBatchInvalid
}

func (s *BookingService) prepareBatchBase(
	ctx context.Context,
	userID int64,
	req dto.BatchBookingRequest,
) (preparedBatchBooking, error) {
	if userID <= 0 || req.ResourceID <= 0 {
		return preparedBatchBooking{}, ErrValidation
	}

	candidates, err := s.normalizeBatchCandidates(req)
	if err != nil {
		return preparedBatchBooking{}, err
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return preparedBatchBooking{}, ErrValidation
		}
		return preparedBatchBooking{}, err
	}
	if !user.IsActive {
		return preparedBatchBooking{}, ErrValidation
	}

	resource, err := s.resources.FindByID(ctx, req.ResourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return preparedBatchBooking{}, ErrResourceNotFound
		}
		return preparedBatchBooking{}, err
	}

	rule, err := s.bookingRules.FindActiveByResourceTypeID(ctx, resource.TypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return preparedBatchBooking{}, ErrBookingRuleNotConfigured
		}
		return preparedBatchBooking{}, err
	}

	activeCount, err := s.bookings.CountByUserAndStatuses(ctx, userID, activeBookingStatuses)
	if err != nil {
		return preparedBatchBooking{}, err
	}

	status := model.BookingStatusConfirmed
	if rule.RequiresApproval {
		status = model.BookingStatusPending
	}

	return preparedBatchBooking{
		Resource:   resource,
		Rule:       rule,
		Status:     status,
		ActiveCount: activeCount,
		Candidates: candidates,
	}, nil
}

func (s *BookingService) normalizeBatchCandidates(req dto.BatchBookingRequest) ([]bookingBatchCandidate, error) {
	if len(req.Dates) == 0 {
		return nil, ErrValidation
	}
	if len(req.Dates) > maxBatchBookingDates {
		return nil, ErrBookingBatchTooLarge
	}

	location := s.appLocation
	if location == nil {
		location = time.UTC
	}

	seen := make(map[string]struct{}, len(req.Dates))
	candidates := make([]bookingBatchCandidate, 0, len(req.Dates))
	for _, rawDate := range req.Dates {
		dateValue := strings.TrimSpace(rawDate)
		if dateValue == "" {
			return nil, ErrValidation
		}
		if _, exists := seen[dateValue]; exists {
			return nil, ErrBookingBatchDuplicateDate
		}
		seen[dateValue] = struct{}{}

		startAt, endAt, err := parseBatchDateAndTimeRange(location, dateValue, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}

		candidates = append(candidates, bookingBatchCandidate{
			Date:    dateValue,
			StartAt: startAt,
			EndAt:   endAt,
		})
	}

	slices.SortFunc(candidates, func(left, right bookingBatchCandidate) int {
		return strings.Compare(left.Date, right.Date)
	})

	return candidates, nil
}

func parseBatchDateAndTimeRange(location *time.Location, dateValue, startTimeValue, endTimeValue string) (time.Time, time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateValue), location)
	if err != nil {
		return time.Time{}, time.Time{}, ErrValidation
	}

	startClock, err := parseWorkdayTime(strings.TrimSpace(startTimeValue))
	if err != nil {
		return time.Time{}, time.Time{}, ErrValidation
	}
	endClock, err := parseWorkdayTime(strings.TrimSpace(endTimeValue))
	if err != nil {
		return time.Time{}, time.Time{}, ErrValidation
	}

	startAt := time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, location)
	endAt := time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 0, 0, location)

	startUTC, endUTC, err := validateBookingRange(startAt.UTC(), endAt.UTC())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return startUTC, endUTC, nil
}

func (s *BookingService) evaluateBatchCandidates(
	ctx context.Context,
	repo repository.BookingRepository,
	prepared preparedBatchBooking,
	now time.Time,
) []dto.BatchBookingPreviewItem {
	items := make([]dto.BatchBookingPreviewItem, 0, len(prepared.Candidates))
	currentMinute := now.UTC().Truncate(time.Minute)
	activeCount := prepared.ActiveCount

	for _, candidate := range prepared.Candidates {
		item := dto.BatchBookingPreviewItem{
			Date:  candidate.Date,
			Valid: false,
		}

		switch {
		case candidate.StartAt.Before(currentMinute):
			item.ErrorCode = stringPtr(batchErrorCodeStartInPast)
		case !prepared.Resource.IsActive || !prepared.Resource.IsBookable:
			item.ErrorCode = stringPtr(batchErrorCodeResourceUnavailable)
		case candidate.EndAt.After(now.AddDate(0, 0, int(prepared.Rule.BookingHorizonDays))):
			item.ErrorCode = stringPtr(batchErrorCodeHorizonExceeded)
		case !s.durationMatchesRule(candidate.StartAt, candidate.EndAt, prepared.Rule):
			item.ErrorCode = stringPtr(dto.ErrorCodeValidation)
		case s.ensureWithinRuleWorkday(candidate.StartAt, candidate.EndAt, prepared.Rule) != nil:
			item.ErrorCode = stringPtr(batchErrorCodeOutsideWorkday)
		default:
			hasUnavailabilityConflict, err := s.unavailability.HasConflict(ctx, prepared.Resource.ID, candidate.StartAt, candidate.EndAt)
			if err != nil {
				item.ErrorCode = stringPtr(dto.ErrorCodeValidation)
				break
			}
			if hasUnavailabilityConflict {
				item.ErrorCode = stringPtr(batchErrorCodeInUnavailability)
				break
			}

			hasConflict, err := repo.HasConflict(ctx, prepared.Resource.ID, candidate.StartAt, candidate.EndAt, activeBookingStatuses)
			if err != nil {
				item.ErrorCode = stringPtr(dto.ErrorCodeValidation)
				break
			}
			if hasConflict {
				item.ErrorCode = stringPtr(batchErrorCodeConflict)
				break
			}

			if activeCount+1 > int64(prepared.Rule.MaxActiveBookingsPerUser) {
				item.ErrorCode = stringPtr(batchErrorCodeLimitExceeded)
				break
			}

			activeCount++
			item.Valid = true
			item.ErrorCode = nil
			item.Status = stringPtr(prepared.Status)
		}

		items = append(items, item)
	}

	return items
}

func (s *BookingService) durationMatchesRule(startAt, endAt time.Time, rule model.BookingRule) bool {
	durationMinutes := endAt.Sub(startAt).Minutes()
	return durationMinutes >= float64(rule.MinDurationMinutes) && durationMinutes <= float64(rule.MaxDurationMinutes)
}

func allBatchItemsValid(items []dto.BatchBookingPreviewItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !item.Valid {
			return false
		}
	}
	return true
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
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
	return s.ListBusyIntervalsByResourceIDInRange(ctx, resourceID, nil, nil)
}

func (s *BookingService) ListBusyIntervalsByResourceIDInRange(
	ctx context.Context,
	resourceID int64,
	from, to *time.Time,
) ([]dto.ResourceBusyIntervalResponse, error) {
	if resourceID <= 0 {
		return nil, ErrValidation
	}

	if _, err := s.resources.FindByID(ctx, resourceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	rangeFrom, rangeTo, err := normalizeBusyIntervalRange(from, to)
	if err != nil {
		return nil, err
	}

	bookings, err := s.bookings.ListBusyIntervalsByResourceID(ctx, resourceID, activeBookingStatuses, rangeFrom, rangeTo)
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

func normalizeBusyIntervalRange(from, to *time.Time) (time.Time, time.Time, error) {
	if from == nil && to == nil {
		now := time.Now().UTC()
		return now, now.AddDate(0, 0, 30), nil
	}

	if from == nil || to == nil {
		return time.Time{}, time.Time{}, ErrBusyIntervalRangeInvalid
	}

	rangeFrom := from.UTC()
	rangeTo := to.UTC()
	if !rangeTo.After(rangeFrom) {
		return time.Time{}, time.Time{}, ErrBusyIntervalRangeInvalid
	}

	if rangeTo.Sub(rangeFrom) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, ErrBusyIntervalRangeInvalid
	}

	return rangeFrom, rangeTo, nil
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
	return s.CancelAt(ctx, id, actorUserID, isPrivileged, time.Now().UTC())
}

func (s *BookingService) CancelAt(
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
		logBookingRejection(ctx, "booking cancel rejected: forbidden foreign booking access",
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
		return dto.BookingResponse{}, errors.Join(ErrBookingForbidden, ErrBookingNotFound)
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

	now = now.UTC()
	if !booking.EndAt.UTC().After(now) {
		logBookingRejection(ctx, "booking cancel rejected: booking has already ended",
			"booking_id", booking.ID,
			"resource_id", booking.ResourceID,
			"booking_user_id", booking.UserID,
			"actor_user_id", actorUserID,
			"status", booking.Status,
			"status_from", booking.Status,
			"status_to", model.BookingStatusCancelled,
			"start_at", booking.StartAt.UTC(),
			"end_at", booking.EndAt.UTC(),
			"now", now,
		)
		return dto.BookingResponse{}, ErrBookingAlreadyEnded
	}

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

func (s *BookingService) ensureWithinRuleWorkday(startAt, endAt time.Time, rule model.BookingRule) error {
	if rule.UnrestrictedTime {
		return nil
	}

	location := s.appLocation
	if location == nil {
		location = time.UTC
	}

	localStart := startAt.In(location)
	localEnd := endAt.In(location)
	if localStart.Year() != localEnd.Year() || localStart.YearDay() != localEnd.YearDay() {
		return ErrBookingOutsideWorkday
	}

	workdayStartValue := effectiveWorkdayTime(rule.WorkdayStart, defaultWorkdayStart)
	workdayEndValue := effectiveWorkdayTime(rule.WorkdayEnd, defaultWorkdayEnd)
	workdayStart := time.Date(
		localStart.Year(),
		localStart.Month(),
		localStart.Day(),
		workdayStartValue.Hour(),
		workdayStartValue.Minute(),
		workdayStartValue.Second(),
		0,
		location,
	)
	workdayEnd := time.Date(
		localStart.Year(),
		localStart.Month(),
		localStart.Day(),
		workdayEndValue.Hour(),
		workdayEndValue.Minute(),
		workdayEndValue.Second(),
		0,
		location,
	)

	if localStart.Before(workdayStart) || localEnd.After(workdayEnd) {
		return ErrBookingOutsideWorkday
	}

	return nil
}

func effectiveWorkdayTime(value time.Time, fallback string) time.Time {
	if !value.IsZero() {
		return value
	}

	parsed, err := parseWorkdayTime(fallback)
	if err != nil {
		return value
	}

	return parsed
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
