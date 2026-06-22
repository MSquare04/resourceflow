package service_test

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
)

var errUnexpectedCall = errors.New("unexpected call")

type bookingRepoMock struct {
	createFn           func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error)
	listFn             func(ctx context.Context) ([]model.Booking, error)
	listByUserIDFn     func(ctx context.Context, userID int64) ([]model.Booking, error)
	listBusyFn         func(ctx context.Context, resourceID int64, statuses []string, from, until time.Time) ([]model.Booking, error)
	findByIDFn         func(ctx context.Context, id int64) (model.Booking, error)
	countByStatusesFn  func(ctx context.Context, userID int64, statuses []string) (int64, error)
	hasConflictFn      func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error)
	isCoveredFn        func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error)
	processExpiredFn   func(ctx context.Context, now time.Time) (repository.ExpiredBookingProcessingResult, error)
	updateStatusFn     func(ctx context.Context, id int64, params repository.UpdateBookingStatusParams) (model.Booking, error)
	transitionStatusFn func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error)
}

func (m *bookingRepoMock) Create(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return model.Booking{}, errUnexpectedCall
}

func (m *bookingRepoMock) List(ctx context.Context) ([]model.Booking, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, errUnexpectedCall
}

func (m *bookingRepoMock) ListByUserID(ctx context.Context, userID int64) ([]model.Booking, error) {
	if m.listByUserIDFn != nil {
		return m.listByUserIDFn(ctx, userID)
	}
	return nil, errUnexpectedCall
}

func (m *bookingRepoMock) ListBusyIntervalsByResourceID(
	ctx context.Context,
	resourceID int64,
	statuses []string,
	from, until time.Time,
) ([]model.Booking, error) {
	if m.listBusyFn != nil {
		return m.listBusyFn(ctx, resourceID, statuses, from, until)
	}
	return nil, errUnexpectedCall
}

func (m *bookingRepoMock) FindByID(ctx context.Context, id int64) (model.Booking, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return model.Booking{}, errUnexpectedCall
}

func (m *bookingRepoMock) CountByUserAndStatuses(ctx context.Context, userID int64, statuses []string) (int64, error) {
	if m.countByStatusesFn != nil {
		return m.countByStatusesFn(ctx, userID, statuses)
	}
	return 0, errUnexpectedCall
}

func (m *bookingRepoMock) HasConflict(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
	if m.hasConflictFn != nil {
		return m.hasConflictFn(ctx, resourceID, startAt, endAt, statuses)
	}
	return false, errUnexpectedCall
}

func (m *bookingRepoMock) IsCoveredByAvailability(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
	if m.isCoveredFn != nil {
		return m.isCoveredFn(ctx, resourceID, startAt, endAt)
	}
	return false, errUnexpectedCall
}

func (m *bookingRepoMock) ProcessExpired(ctx context.Context, now time.Time) (repository.ExpiredBookingProcessingResult, error) {
	if m.processExpiredFn != nil {
		return m.processExpiredFn(ctx, now)
	}
	return repository.ExpiredBookingProcessingResult{}, errUnexpectedCall
}

func (m *bookingRepoMock) UpdateStatus(ctx context.Context, id int64, params repository.UpdateBookingStatusParams) (model.Booking, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, params)
	}
	return model.Booking{}, errUnexpectedCall
}

func (m *bookingRepoMock) TransitionStatus(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, id, expectedFrom, params)
	}
	return model.Booking{}, "", errUnexpectedCall
}

type resourceRepoMock struct {
	createFn   func(ctx context.Context, params repository.CreateResourceParams) (model.Resource, error)
	listFn     func(ctx context.Context) ([]model.Resource, error)
	findByIDFn func(ctx context.Context, id int64) (model.Resource, error)
	updateFn   func(ctx context.Context, id int64, params repository.UpdateResourceParams) (model.Resource, error)
}

func (m *resourceRepoMock) Create(ctx context.Context, params repository.CreateResourceParams) (model.Resource, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return model.Resource{}, errUnexpectedCall
}

func (m *resourceRepoMock) List(ctx context.Context) ([]model.Resource, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, errUnexpectedCall
}

func (m *resourceRepoMock) FindByID(ctx context.Context, id int64) (model.Resource, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return model.Resource{}, errUnexpectedCall
}

func (m *resourceRepoMock) Update(ctx context.Context, id int64, params repository.UpdateResourceParams) (model.Resource, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, params)
	}
	return model.Resource{}, errUnexpectedCall
}

type userRepoMock struct {
	findByEmailFn      func(ctx context.Context, email string) (model.User, error)
	findByIDFn         func(ctx context.Context, id int64) (model.User, error)
	createFn           func(ctx context.Context, params repository.CreateUserParams) (model.User, error)
	listFn             func(ctx context.Context) ([]model.User, error)
	updateFn           func(ctx context.Context, id int64, params repository.UpdateUserParams) (model.User, error)
	listRolesByUserFn  func(ctx context.Context, userID int64) ([]string, error)
	validateRoleCodes  func(ctx context.Context, roleCodes []string) error
	replaceRolesByUser func(ctx context.Context, userID int64, roleCodes []string) error
}

func (m *userRepoMock) FindByEmail(ctx context.Context, email string) (model.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return model.User{}, errUnexpectedCall
}

func (m *userRepoMock) FindByID(ctx context.Context, id int64) (model.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return model.User{}, errUnexpectedCall
}

func (m *userRepoMock) Create(ctx context.Context, params repository.CreateUserParams) (model.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return model.User{}, errUnexpectedCall
}

func (m *userRepoMock) List(ctx context.Context) ([]model.User, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, errUnexpectedCall
}

func (m *userRepoMock) Update(ctx context.Context, id int64, params repository.UpdateUserParams) (model.User, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, params)
	}
	return model.User{}, errUnexpectedCall
}

func (m *userRepoMock) ListRolesByUserID(ctx context.Context, userID int64) ([]string, error) {
	if m.listRolesByUserFn != nil {
		return m.listRolesByUserFn(ctx, userID)
	}
	return nil, errUnexpectedCall
}

func (m *userRepoMock) ValidateRoleCodes(ctx context.Context, roleCodes []string) error {
	if m.validateRoleCodes != nil {
		return m.validateRoleCodes(ctx, roleCodes)
	}
	return nil
}

func (m *userRepoMock) ReplaceRolesByUserID(ctx context.Context, userID int64, roleCodes []string) error {
	if m.replaceRolesByUser != nil {
		return m.replaceRolesByUser(ctx, userID, roleCodes)
	}
	return nil
}

type bookingRuleRepoMock struct {
	createFn                     func(ctx context.Context, params repository.CreateBookingRuleParams) (model.BookingRule, error)
	listFn                       func(ctx context.Context) ([]model.BookingRule, error)
	findByIDFn                   func(ctx context.Context, id int64) (model.BookingRule, error)
	findActiveByResourceTypeIDFn func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error)
	updateFn                     func(ctx context.Context, id int64, params repository.UpdateBookingRuleParams) (model.BookingRule, error)
}

func (m *bookingRuleRepoMock) Create(ctx context.Context, params repository.CreateBookingRuleParams) (model.BookingRule, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return model.BookingRule{}, errUnexpectedCall
}

func (m *bookingRuleRepoMock) List(ctx context.Context) ([]model.BookingRule, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, errUnexpectedCall
}

func (m *bookingRuleRepoMock) FindByID(ctx context.Context, id int64) (model.BookingRule, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return model.BookingRule{}, errUnexpectedCall
}

func (m *bookingRuleRepoMock) FindActiveByResourceTypeID(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
	if m.findActiveByResourceTypeIDFn != nil {
		return m.findActiveByResourceTypeIDFn(ctx, resourceTypeID)
	}
	return model.BookingRule{}, errUnexpectedCall
}

func (m *bookingRuleRepoMock) Update(ctx context.Context, id int64, params repository.UpdateBookingRuleParams) (model.BookingRule, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, params)
	}
	return model.BookingRule{}, errUnexpectedCall
}

type resourceTypeRepoMock struct {
	createFn              func(ctx context.Context, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error)
	listFn                func(ctx context.Context) ([]model.ResourceType, error)
	findByIDFn            func(ctx context.Context, id int64) (model.ResourceType, error)
	existsByIDAndCategory func(ctx context.Context, id int64, categoryID int64) (bool, error)
	updateFn              func(ctx context.Context, id int64, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error)
}

func (m *resourceTypeRepoMock) Create(ctx context.Context, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error) {
	if m.createFn != nil {
		return m.createFn(ctx, categoryID, code, name, description, isActive)
	}
	return model.ResourceType{}, errUnexpectedCall
}

func (m *resourceTypeRepoMock) List(ctx context.Context) ([]model.ResourceType, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, errUnexpectedCall
}

func (m *resourceTypeRepoMock) FindByID(ctx context.Context, id int64) (model.ResourceType, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return model.ResourceType{}, errUnexpectedCall
}

func (m *resourceTypeRepoMock) ExistsByIDAndCategory(ctx context.Context, id int64, categoryID int64) (bool, error) {
	if m.existsByIDAndCategory != nil {
		return m.existsByIDAndCategory(ctx, id, categoryID)
	}
	return false, errUnexpectedCall
}

func (m *resourceTypeRepoMock) Update(ctx context.Context, id int64, categoryID int64, code, name, description string, isActive bool) (model.ResourceType, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, categoryID, code, name, description, isActive)
	}
	return model.ResourceType{}, errUnexpectedCall
}

type availabilityRepoMock struct {
	createFn              func(ctx context.Context, params repository.CreateResourceAvailabilityParams) (model.ResourceAvailability, error)
	listByResourceIDFn    func(ctx context.Context, resourceID int64) ([]model.ResourceAvailability, error)
	findByIDAndResourceFn func(ctx context.Context, resourceID int64, id int64) (model.ResourceAvailability, error)
	updateFn              func(ctx context.Context, resourceID int64, id int64, params repository.UpdateResourceAvailabilityParams) (model.ResourceAvailability, error)
	deleteFn              func(ctx context.Context, resourceID int64, id int64) (bool, error)
}

func (m *availabilityRepoMock) Create(ctx context.Context, params repository.CreateResourceAvailabilityParams) (model.ResourceAvailability, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return model.ResourceAvailability{}, errUnexpectedCall
}

func (m *availabilityRepoMock) ListByResourceID(ctx context.Context, resourceID int64) ([]model.ResourceAvailability, error) {
	if m.listByResourceIDFn != nil {
		return m.listByResourceIDFn(ctx, resourceID)
	}
	return nil, errUnexpectedCall
}

func (m *availabilityRepoMock) FindByIDAndResourceID(ctx context.Context, resourceID int64, id int64) (model.ResourceAvailability, error) {
	if m.findByIDAndResourceFn != nil {
		return m.findByIDAndResourceFn(ctx, resourceID, id)
	}
	return model.ResourceAvailability{}, sql.ErrNoRows
}

func (m *availabilityRepoMock) Update(ctx context.Context, resourceID int64, id int64, params repository.UpdateResourceAvailabilityParams) (model.ResourceAvailability, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, resourceID, id, params)
	}
	return model.ResourceAvailability{}, errUnexpectedCall
}

func (m *availabilityRepoMock) Delete(ctx context.Context, resourceID int64, id int64) (bool, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, resourceID, id)
	}
	return false, errUnexpectedCall
}
