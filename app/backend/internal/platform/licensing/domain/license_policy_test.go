package domain

import (
	"testing"

	apperrors "github.com/links/backend/pkg/shared/errors"
)

func TestValidateSelfServiceSeatsRejectsAboveLimit(t *testing.T) {
	err := NewLicensePolicy().ValidateSelfServiceSeats(MaxSelfServiceSeats+1, 0)
	if code, ok := apperrors.CodeOf(err); !ok || code != apperrors.CodeInvalidArgument {
		t.Fatalf("error code = %v, %v; want invalid_argument", code, ok)
	}
}

func TestValidateSelfServiceSeatsRejectsBelowUsedSeats(t *testing.T) {
	err := NewLicensePolicy().ValidateSelfServiceSeats(2, 3)
	if code, ok := apperrors.CodeOf(err); !ok || code != apperrors.CodeProductSeatsBelowUse {
		t.Fatalf("error code = %v, %v; want product_seats_below_use", code, ok)
	}
}

func TestEnsureAssignableSeatAllowsAlreadyActiveAssignment(t *testing.T) {
	err := NewLicensePolicy().EnsureAssignableSeat("active", true, 3, 3)
	if err != nil {
		t.Fatalf("EnsureAssignableSeat() error = %v", err)
	}
}

func TestEnsureAssignableSeatRejectsSeatLimit(t *testing.T) {
	err := NewLicensePolicy().EnsureAssignableSeat("removed", true, 3, 3)
	if code, ok := apperrors.CodeOf(err); !ok || code != apperrors.CodeProductSeatLimit {
		t.Fatalf("error code = %v, %v; want product_seat_limit", code, ok)
	}
}

func TestChangeType(t *testing.T) {
	policy := NewLicensePolicy()
	cases := []struct {
		name     string
		existing bool
		before   int32
		after    int32
		want     string
	}{
		{name: "new", existing: false, before: 0, after: 2, want: "granted"},
		{name: "increase", existing: true, before: 1, after: 2, want: "increased"},
		{name: "decrease", existing: true, before: 2, after: 1, want: "decreased"},
		{name: "same", existing: true, before: 2, after: 2, want: "renewed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := policy.ChangeType(tc.existing, tc.before, tc.after); got != tc.want {
				t.Fatalf("ChangeType() = %q, want %q", got, tc.want)
			}
		})
	}
}
