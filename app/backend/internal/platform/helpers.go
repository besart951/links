package platform

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	apperrors "github.com/links/backend/pkg/shared/errors"
)

func uniqueTenantSlug(ctx context.Context, tx pgx.Tx, name string) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "workspace"
	}
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tenants WHERE slug = $1)`, candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("tenant slug exhausted")
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlug.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "SQLSTATE 23505")
}

func invalid(message string) error {
	return apperrors.Wrap(apperrors.CodeInvalidArgument, errors.New(message))
}

func internal(err error) error {
	return apperrors.Wrap(apperrors.CodeInternal, err)
}
