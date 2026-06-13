package platform

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	entitlements "github.com/links/backend/internal/platform/entitlements/domain"
	apperrors "github.com/links/backend/pkg/shared/errors"
)

func (s *Service) ListProductAssignments(ctx context.Context, session *SessionRow, productKey string) (*backendv1.ListProductAssignmentsResponse, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleAdmin); err != nil {
		return nil, err
	}
	args := []any{session.ActiveTenantID}
	filter := ""
	if productKey != "" {
		args = append(args, productKey)
		filter = " AND pa.product_key = $2"
	}
	rows, err := s.pool.Query(ctx, `
		SELECT pa.id::text, pa.tenant_id::text, pa.user_id::text, u.email, u.display_name,
		       pa.product_key, p.name, pa.role, pa.status, pa.assigned_at
		FROM product_assignments pa
		JOIN users u ON u.id = pa.user_id
		JOIN products p ON p.key = pa.product_key
		WHERE pa.tenant_id = $1`+filter+`
		ORDER BY p.key, u.email
	`, args...)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var assignments []*backendv1.ProductAssignment
	for rows.Next() {
		assignment, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	if rows.Err() != nil {
		return nil, internal(rows.Err())
	}
	return &backendv1.ListProductAssignmentsResponse{Assignments: assignments}, nil
}

func (s *Service) AssignUserToProduct(ctx context.Context, session *SessionRow, userID, productKey, role string) (*backendv1.ProductAssignment, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleAdmin); err != nil {
		return nil, err
	}
	if role == "" {
		role = string(entitlements.ProductRoleMember)
	}
	if !s.accessPolicy.ValidProductRole(entitlements.ProductRole(role)) {
		return nil, invalid("INVALID_PRODUCT_ROLE")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, internal(err)
	}
	defer rollback(ctx, tx)

	var seatsTotal int32
	if err := tx.QueryRow(ctx, `
		SELECT seats_total
		FROM product_license_pools
		WHERE tenant_id = $1 AND product_key = $2 AND status IN ('active', 'trial')
		FOR UPDATE
	`, session.ActiveTenantID, productKey).Scan(&seatsTotal); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.CodeProductNotLicensed)
		}
		return nil, internal(err)
	}
	var isMember bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM tenant_memberships
			WHERE tenant_id = $1 AND user_id = $2 AND status IN ('active', 'invited')
		)
	`, session.ActiveTenantID, userID).Scan(&isMember); err != nil {
		return nil, internal(err)
	}
	if !isMember {
		return nil, apperrors.Wrap(apperrors.CodeFailedPrecondition, errors.New("USER_NOT_TENANT_MEMBER"))
	}

	var currentStatus sql.NullString
	err = tx.QueryRow(ctx, `
		SELECT status
		FROM product_assignments
		WHERE tenant_id = $1 AND user_id = $2 AND product_key = $3
	`, session.ActiveTenantID, userID, productKey).Scan(&currentStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, internal(err)
	}
	var seatsUsed int32
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM product_assignments
		WHERE tenant_id = $1 AND product_key = $2 AND status = 'active'
	`, session.ActiveTenantID, productKey).Scan(&seatsUsed); err != nil {
		return nil, internal(err)
	}
	if err := s.licensePolicy.EnsureAssignableSeat(currentStatus.String, currentStatus.Valid, seatsUsed, seatsTotal); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO product_assignments (tenant_id, user_id, product_key, role, status, assigned_by, assigned_at, removed_at)
		VALUES ($1, $2, $3, $4, 'active', $5, now(), NULL)
		ON CONFLICT (tenant_id, user_id, product_key)
		DO UPDATE SET role = EXCLUDED.role, status = 'active', assigned_by = EXCLUDED.assigned_by, assigned_at = now(), removed_at = NULL
	`, session.ActiveTenantID, userID, productKey, role, session.UserID); err != nil {
		return nil, internal(err)
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'product.assignment.created', 'user', $3)
	`, session.UserID, session.ActiveTenantID, userID)
	if err := tx.Commit(ctx); err != nil {
		return nil, internal(err)
	}
	return s.getAssignment(ctx, session.ActiveTenantID, userID, productKey)
}

func (s *Service) RemoveUserFromProduct(ctx context.Context, session *SessionRow, userID, productKey string) (*backendv1.ProductAssignment, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleAdmin); err != nil {
		return nil, err
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE product_assignments
		SET status = 'removed', removed_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND product_key = $3
	`, session.ActiveTenantID, userID, productKey); err != nil {
		return nil, internal(err)
	}
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'product.assignment.removed', 'user', $3)
	`, session.UserID, session.ActiveTenantID, userID)
	return s.getAssignment(ctx, session.ActiveTenantID, userID, productKey)
}

func (s *Service) UpdateProductRole(ctx context.Context, session *SessionRow, userID, productKey, role string) (*backendv1.ProductAssignment, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleAdmin); err != nil {
		return nil, err
	}
	if !s.accessPolicy.ValidProductRole(entitlements.ProductRole(role)) {
		return nil, invalid("INVALID_PRODUCT_ROLE")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE product_assignments
		SET role = $1
		WHERE tenant_id = $2 AND user_id = $3 AND product_key = $4 AND status = 'active'
	`, role, session.ActiveTenantID, userID, productKey)
	if err != nil {
		return nil, internal(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperrors.New(apperrors.CodeAssignmentNotFound)
	}
	return s.getAssignment(ctx, session.ActiveTenantID, userID, productKey)
}

func (s *Service) RequireProductPermission(ctx context.Context, session *SessionRow, productKey entitlements.ProductKey, permission entitlements.Permission) error {
	access, err := s.BuildAccessSnapshot(ctx, session.UserID, session.ActiveTenantID)
	if err != nil {
		return err
	}
	for _, product := range access.Products {
		if product.Key != string(productKey) || !product.Assigned || product.Status != "active" {
			continue
		}
		for _, candidate := range product.Permissions {
			if candidate == string(permission) {
				return nil
			}
		}
	}
	return apperrors.New(apperrors.CodePermissionDenied)
}

func (s *Service) getAssignment(ctx context.Context, tenantID, userID, productKey string) (*backendv1.ProductAssignment, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT pa.id::text, pa.tenant_id::text, pa.user_id::text, u.email, u.display_name,
		       pa.product_key, p.name, pa.role, pa.status, pa.assigned_at
		FROM product_assignments pa
		JOIN users u ON u.id = pa.user_id
		JOIN products p ON p.key = pa.product_key
		WHERE pa.tenant_id = $1 AND pa.user_id = $2 AND pa.product_key = $3
	`, tenantID, userID, productKey)
	assignment, err := scanAssignment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.CodeAssignmentNotFound)
		}
		return nil, err
	}
	return assignment, nil
}

type assignmentScanner interface {
	Scan(dest ...any) error
}

func scanAssignment(row assignmentScanner) (*backendv1.ProductAssignment, error) {
	assignment := &backendv1.ProductAssignment{}
	var assignedAt time.Time
	if err := row.Scan(
		&assignment.Id,
		&assignment.TenantId,
		&assignment.UserId,
		&assignment.Email,
		&assignment.DisplayName,
		&assignment.ProductKey,
		&assignment.ProductName,
		&assignment.Role,
		&assignment.Status,
		&assignedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, internal(err)
	}
	assignment.AssignedAt = assignedAt.Format(time.RFC3339)
	return assignment, nil
}
