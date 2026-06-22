package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
)

var ErrBookingRuleNotFound = errors.New("booking rule not found")

const (
	defaultWorkdayStart = "07:00"
	defaultWorkdayEnd   = "22:00"
	workdayTimeLayout   = "15:04"
)

type BookingRuleService struct {
	bookingRules  repository.BookingRuleRepository
	resourceTypes repository.ResourceTypeRepository
}

func NewBookingRuleService(
	bookingRules repository.BookingRuleRepository,
	resourceTypes repository.ResourceTypeRepository,
) *BookingRuleService {
	return &BookingRuleService{
		bookingRules:  bookingRules,
		resourceTypes: resourceTypes,
	}
}

func (s *BookingRuleService) Create(ctx context.Context, req dto.CreateBookingRuleRequest) (dto.BookingRuleResponse, error) {
	if err := validateBookingRulePayload(
		req.ResourceTypeID,
		req.MinDurationMinutes,
		req.MaxDurationMinutes,
		req.MaxActiveBookingsPerUser,
		req.BookingHorizonDays,
	); err != nil {
		return dto.BookingRuleResponse{}, err
	}

	if err := s.ensureResourceTypeExists(ctx, req.ResourceTypeID); err != nil {
		return dto.BookingRuleResponse{}, err
	}

	requiresApproval := false
	if req.RequiresApproval != nil {
		requiresApproval = *req.RequiresApproval
	}

	unrestrictedTime := false
	if req.UnrestrictedTime != nil {
		unrestrictedTime = *req.UnrestrictedTime
	}

	workdayStart, workdayEnd, err := resolveWorkdayWindow(
		req.WorkdayStart,
		req.WorkdayEnd,
		nil,
		nil,
		unrestrictedTime,
	)
	if err != nil {
		return dto.BookingRuleResponse{}, err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	rule, err := s.bookingRules.Create(ctx, repository.CreateBookingRuleParams{
		ResourceTypeID:           req.ResourceTypeID,
		MinDurationMinutes:       req.MinDurationMinutes,
		MaxDurationMinutes:       req.MaxDurationMinutes,
		MaxActiveBookingsPerUser: req.MaxActiveBookingsPerUser,
		RequiresApproval:         requiresApproval,
		BookingHorizonDays:       req.BookingHorizonDays,
		WorkdayStart:             workdayStart,
		WorkdayEnd:               workdayEnd,
		UnrestrictedTime:         unrestrictedTime,
		IsActive:                 isActive,
	})
	if err != nil {
		if isForeignKeyViolation(err) || isCheckViolation(err) {
			return dto.BookingRuleResponse{}, ErrValidation
		}
		return dto.BookingRuleResponse{}, err
	}

	return mapBookingRuleResponse(rule), nil
}

func (s *BookingRuleService) List(ctx context.Context) ([]dto.BookingRuleResponse, error) {
	rules, err := s.bookingRules.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.BookingRuleResponse, 0, len(rules))
	for _, rule := range rules {
		result = append(result, mapBookingRuleResponse(rule))
	}

	return result, nil
}

func (s *BookingRuleService) GetByID(ctx context.Context, id int64) (dto.BookingRuleResponse, error) {
	if id <= 0 {
		return dto.BookingRuleResponse{}, ErrValidation
	}

	rule, err := s.bookingRules.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.BookingRuleResponse{}, ErrBookingRuleNotFound
		}
		return dto.BookingRuleResponse{}, err
	}

	return mapBookingRuleResponse(rule), nil
}

func (s *BookingRuleService) Update(ctx context.Context, id int64, req dto.UpdateBookingRuleRequest) (dto.BookingRuleResponse, error) {
	if id <= 0 {
		return dto.BookingRuleResponse{}, ErrValidation
	}

	if err := validateBookingRulePayload(
		req.ResourceTypeID,
		req.MinDurationMinutes,
		req.MaxDurationMinutes,
		req.MaxActiveBookingsPerUser,
		req.BookingHorizonDays,
	); err != nil {
		return dto.BookingRuleResponse{}, err
	}

	current, err := s.bookingRules.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.BookingRuleResponse{}, ErrBookingRuleNotFound
		}
		return dto.BookingRuleResponse{}, err
	}

	if err := s.ensureResourceTypeExists(ctx, req.ResourceTypeID); err != nil {
		return dto.BookingRuleResponse{}, err
	}

	requiresApproval := current.RequiresApproval
	if req.RequiresApproval != nil {
		requiresApproval = *req.RequiresApproval
	}

	unrestrictedTime := current.UnrestrictedTime
	if req.UnrestrictedTime != nil {
		unrestrictedTime = *req.UnrestrictedTime
	}

	workdayStart, workdayEnd, err := resolveWorkdayWindow(
		req.WorkdayStart,
		req.WorkdayEnd,
		&current.WorkdayStart,
		&current.WorkdayEnd,
		unrestrictedTime,
	)
	if err != nil {
		return dto.BookingRuleResponse{}, err
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	rule, err := s.bookingRules.Update(ctx, id, repository.UpdateBookingRuleParams{
		ResourceTypeID:           req.ResourceTypeID,
		MinDurationMinutes:       req.MinDurationMinutes,
		MaxDurationMinutes:       req.MaxDurationMinutes,
		MaxActiveBookingsPerUser: req.MaxActiveBookingsPerUser,
		RequiresApproval:         requiresApproval,
		BookingHorizonDays:       req.BookingHorizonDays,
		WorkdayStart:             workdayStart,
		WorkdayEnd:               workdayEnd,
		UnrestrictedTime:         unrestrictedTime,
		IsActive:                 isActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return dto.BookingRuleResponse{}, ErrBookingRuleNotFound
		case isForeignKeyViolation(err), isCheckViolation(err):
			return dto.BookingRuleResponse{}, ErrValidation
		default:
			return dto.BookingRuleResponse{}, err
		}
	}

	return mapBookingRuleResponse(rule), nil
}

func (s *BookingRuleService) ensureResourceTypeExists(ctx context.Context, resourceTypeID int64) error {
	if resourceTypeID <= 0 {
		return ErrValidation
	}

	_, err := s.resourceTypes.FindByID(ctx, resourceTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrValidation
		}
		return err
	}

	return nil
}

func validateBookingRulePayload(
	resourceTypeID int64,
	minDurationMinutes int32,
	maxDurationMinutes int32,
	maxActiveBookingsPerUser int32,
	bookingHorizonDays int32,
) error {
	if resourceTypeID <= 0 {
		return ErrValidation
	}
	if minDurationMinutes <= 0 {
		return ErrValidation
	}
	if maxDurationMinutes < minDurationMinutes {
		return ErrValidation
	}
	if maxActiveBookingsPerUser < 1 {
		return ErrValidation
	}
	if bookingHorizonDays < 0 {
		return ErrValidation
	}

	return nil
}

func mapBookingRuleResponse(rule model.BookingRule) dto.BookingRuleResponse {
	return dto.BookingRuleResponse{
		ID:                       rule.ID,
		ResourceTypeID:           rule.ResourceTypeID,
		MinDurationMinutes:       rule.MinDurationMinutes,
		MaxDurationMinutes:       rule.MaxDurationMinutes,
		MaxActiveBookingsPerUser: rule.MaxActiveBookingsPerUser,
		RequiresApproval:         rule.RequiresApproval,
		BookingHorizonDays:       rule.BookingHorizonDays,
		WorkdayStart:             formatWorkdayTime(rule.WorkdayStart),
		WorkdayEnd:               formatWorkdayTime(rule.WorkdayEnd),
		UnrestrictedTime:         rule.UnrestrictedTime,
		IsActive:                 rule.IsActive,
		CreatedAt:                rule.CreatedAt.UTC(),
		UpdatedAt:                rule.UpdatedAt.UTC(),
	}
}

func resolveWorkdayWindow(
	requestStart *string,
	requestEnd *string,
	currentStart *time.Time,
	currentEnd *time.Time,
	unrestricted bool,
) (time.Time, time.Time, error) {
	startValue := defaultWorkdayStart
	if currentStart != nil {
		startValue = formatWorkdayTime(*currentStart)
	}
	if requestStart != nil {
		startValue = strings.TrimSpace(*requestStart)
	}

	endValue := defaultWorkdayEnd
	if currentEnd != nil {
		endValue = formatWorkdayTime(*currentEnd)
	}
	if requestEnd != nil {
		endValue = strings.TrimSpace(*requestEnd)
	}

	startTime, err := parseWorkdayTime(startValue)
	if err != nil {
		return time.Time{}, time.Time{}, ErrValidation
	}
	endTime, err := parseWorkdayTime(endValue)
	if err != nil {
		return time.Time{}, time.Time{}, ErrValidation
	}

	if !unrestricted && !startTime.Before(endTime) {
		return time.Time{}, time.Time{}, ErrValidation
	}

	return startTime, endTime, nil
}

func parseWorkdayTime(value string) (time.Time, error) {
	return time.Parse(workdayTimeLayout, strings.TrimSpace(value))
}

func formatWorkdayTime(value time.Time) string {
	return value.Format(workdayTimeLayout)
}
