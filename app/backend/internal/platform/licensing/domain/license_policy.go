package domain

import (
	"fmt"

	apperrors "github.com/links/backend/pkg/shared/errors"
)

const MaxSelfServiceSeats int32 = 10

type LicensePolicy struct {
	maxSelfServiceSeats int32
}

func NewLicensePolicy() LicensePolicy {
	return LicensePolicy{maxSelfServiceSeats: MaxSelfServiceSeats}
}

func (p LicensePolicy) ValidateSelfServiceSeats(seatsTotal, seatsUsed int32) error {
	if seatsTotal < 0 || seatsTotal > p.maxSelfServiceSeats {
		return apperrors.Wrap(apperrors.CodeInvalidArgument, fmt.Errorf("PRODUCT_SEAT_LIMIT_MAX_%d", p.maxSelfServiceSeats))
	}
	if seatsTotal < seatsUsed {
		return apperrors.New(apperrors.CodeProductSeatsBelowUse)
	}
	return nil
}

func (LicensePolicy) EnsureAssignableSeat(currentStatus string, hasExistingAssignment bool, seatsUsed, seatsTotal int32) error {
	if hasExistingAssignment && currentStatus == "active" {
		return nil
	}
	if seatsUsed >= seatsTotal {
		return apperrors.New(apperrors.CodeProductSeatLimit)
	}
	return nil
}

func (LicensePolicy) ChangeType(hasExistingPool bool, before, after int32) string {
	if !hasExistingPool {
		return "granted"
	}
	if after > before {
		return "increased"
	}
	if after < before {
		return "decreased"
	}
	return "renewed"
}
