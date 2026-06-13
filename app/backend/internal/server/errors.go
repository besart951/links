package server

import (
	"errors"

	"connectrpc.com/connect"
	apperrors "github.com/links/backend/pkg/shared/errors"
)

func rpcError(err error) error {
	if err == nil {
		return nil
	}
	code, ok := apperrors.CodeOf(err)
	if !ok {
		return connect.NewError(connect.CodeInternal, err)
	}
	switch code {
	case apperrors.CodeAlreadyExists, apperrors.CodeEmailAlreadyExists:
		return connect.NewError(connect.CodeAlreadyExists, errors.New(string(code)))
	case apperrors.CodeInvalidArgument:
		return connect.NewError(connect.CodeInvalidArgument, errors.New(string(code)))
	case apperrors.CodeUnauthenticated, apperrors.CodeInvalidCredentials, apperrors.CodeUserNotActive, apperrors.CodeUserDisabled, apperrors.CodeSessionExpired:
		return connect.NewError(connect.CodeUnauthenticated, errors.New(string(code)))
	case apperrors.CodePermissionDenied, apperrors.CodeTenantAccessDenied:
		return connect.NewError(connect.CodePermissionDenied, errors.New(string(code)))
	case apperrors.CodeNotFound, apperrors.CodeProductNotFound, apperrors.CodeAssignmentNotFound, apperrors.CodeInviteNotFound:
		return connect.NewError(connect.CodeNotFound, errors.New(string(code)))
	case apperrors.CodeFailedPrecondition, apperrors.CodeNoActiveTenant, apperrors.CodeProductNotLicensed, apperrors.CodeProductSeatsBelowUse, apperrors.CodeProductSeatLimit:
		return connect.NewError(connect.CodeFailedPrecondition, errors.New(string(code)))
	default:
		return connect.NewError(connect.CodeInternal, errors.New(string(code)))
	}
}
