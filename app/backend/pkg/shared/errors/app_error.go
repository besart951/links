package errors

import (
	"errors"
	"fmt"
)

type Code string

const (
	CodeAlreadyExists        Code = "ALREADY_EXISTS"
	CodeInvalidArgument      Code = "INVALID_ARGUMENT"
	CodeUnauthenticated      Code = "UNAUTHENTICATED"
	CodePermissionDenied     Code = "PERMISSION_DENIED"
	CodeNotFound             Code = "NOT_FOUND"
	CodeFailedPrecondition   Code = "FAILED_PRECONDITION"
	CodeInternal             Code = "INTERNAL"
	CodeEmailAlreadyExists   Code = "EMAIL_ALREADY_EXISTS"
	CodeInvalidCredentials   Code = "INVALID_CREDENTIALS"
	CodeUserNotActive        Code = "USER_NOT_ACTIVE"
	CodeUserDisabled         Code = "USER_DISABLED"
	CodeSessionExpired       Code = "SESSION_EXPIRED"
	CodeNoActiveTenant       Code = "NO_ACTIVE_TENANT"
	CodeTenantAccessDenied   Code = "TENANT_ACCESS_DENIED"
	CodeProductNotFound      Code = "PRODUCT_NOT_FOUND"
	CodeProductNotLicensed   Code = "PRODUCT_NOT_LICENSED"
	CodeProductSeatsBelowUse Code = "PRODUCT_SEATS_BELOW_USED"
	CodeProductSeatLimit     Code = "PRODUCT_SEAT_LIMIT_REACHED"
	CodeAssignmentNotFound   Code = "PRODUCT_ASSIGNMENT_NOT_FOUND"
	CodeInviteNotFound       Code = "INVITE_NOT_FOUND"
)

type AppError struct {
	code Code
	err  error
}

func New(code Code) *AppError {
	return &AppError{code: code, err: errors.New(string(code))}
}

func Wrap(code Code, err error) *AppError {
	if err == nil {
		return New(code)
	}
	return &AppError{code: code, err: err}
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %v", e.code, e.err)
}

func (e *AppError) Unwrap() error {
	return e.err
}

func (e *AppError) Code() Code {
	return e.code
}

func CodeOf(err error) (Code, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code(), true
	}
	return "", false
}
