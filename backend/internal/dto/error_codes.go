package dto

const (
	ErrorCodeValidation                = "validation_error"
	ErrorCodeUnauthorized              = "unauthorized"
	ErrorCodeForbidden                 = "forbidden"
	ErrorCodeInactiveUser              = "inactive_user"
	ErrorCodeCurrentPasswordInvalid    = "current_password_invalid"
	ErrorCodeNewPasswordSameAsCurrent  = "new_password_same_as_current"
	ErrorCodePasswordPolicyViolation   = "password_policy_violation"
	ErrorCodeBookingTooEarlyToComplete = "booking_too_early_to_complete"
	ErrorCodeBookingForbidden          = "booking_forbidden"
	ErrorCodeBookingAlreadyEnded       = "booking_already_ended"
	ErrorCodeBookingCancelNotAllowed   = "booking_cancel_not_allowed"
	ErrorCodeBookingOutsideWorkday     = "booking_outside_workday"
	ErrorCodeBookingInUnavailability   = "booking_in_unavailability"
	ErrorCodeNotFound                  = "not_found"
	ErrorCodeConflict                  = "conflict"
	ErrorCodeInternal                  = "internal_error"
)
