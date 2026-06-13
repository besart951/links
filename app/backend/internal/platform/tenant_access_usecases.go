package platform

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	entitlements "github.com/links/backend/internal/platform/entitlements/domain"
	apperrors "github.com/links/backend/pkg/shared/errors"
)

func (s *Service) ListMyTenants(ctx context.Context, session *SessionRow) (*backendv1.ListMyTenantsResponse, error) {
	tenants, err := s.listTenants(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	return &backendv1.ListMyTenantsResponse{Tenants: tenants}, nil
}

func (s *Service) SwitchTenant(ctx context.Context, session *SessionRow, tenantID string) (*backendv1.AccessSnapshotResponse, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM tenant_memberships tm
			JOIN tenants t ON t.id = tm.tenant_id
			WHERE tm.user_id = $1 AND tm.tenant_id = $2
			  AND tm.status = 'active' AND t.status = 'active'
		)
	`, session.UserID, tenantID).Scan(&ok)
	if err != nil {
		return nil, internal(err)
	}
	if !ok {
		return nil, apperrors.New(apperrors.CodeTenantAccessDenied)
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE sessions SET active_tenant_id = $1 WHERE id = $2
	`, tenantID, session.ID); err != nil {
		return nil, internal(err)
	}
	access, err := s.BuildAccessSnapshot(ctx, session.UserID, tenantID)
	if err != nil {
		return nil, err
	}
	return &backendv1.AccessSnapshotResponse{Access: access}, nil
}

func (s *Service) BuildAccessSnapshot(ctx context.Context, userID, activeTenantID string) (*backendv1.AccessSnapshot, error) {
	var user backendv1.AccessUser
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email, display_name, platform_role
		FROM users
		WHERE id = $1 AND status = 'active'
	`, userID).Scan(&user.Id, &user.Email, &user.DisplayName, &user.PlatformRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.CodeUserDisabled)
		}
		return nil, internal(err)
	}
	tenants, err := s.listTenants(ctx, userID)
	if err != nil {
		return nil, err
	}
	var active backendv1.ActiveTenant
	err = s.pool.QueryRow(ctx, `
		SELECT t.id::text, t.name, t.slug, tm.role
		FROM tenants t
		JOIN tenant_memberships tm ON tm.tenant_id = t.id
		WHERE t.id = $1 AND tm.user_id = $2 AND t.status = 'active' AND tm.status = 'active'
	`, activeTenantID, userID).Scan(&active.Id, &active.Name, &active.Slug, &active.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.CodeTenantAccessDenied)
		}
		return nil, internal(err)
	}
	products, err := s.productAccess(ctx, userID, activeTenantID)
	if err != nil {
		return nil, err
	}
	return &backendv1.AccessSnapshot{
		User:         &user,
		ActiveTenant: &active,
		Tenants:      tenants,
		Products:     products,
	}, nil
}

func (s *Service) firstActiveTenantID(ctx context.Context, userID string) (string, error) {
	var tenantID string
	err := s.pool.QueryRow(ctx, `
		SELECT t.id::text
		FROM tenant_memberships tm
		JOIN tenants t ON t.id = tm.tenant_id
		WHERE tm.user_id = $1 AND tm.status = 'active' AND t.status = 'active'
		ORDER BY tm.created_at ASC
		LIMIT 1
	`, userID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperrors.New(apperrors.CodeNoActiveTenant)
		}
		return "", internal(err)
	}
	return tenantID, nil
}

func (s *Service) listTenants(ctx context.Context, userID string) ([]*backendv1.TenantSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id::text, t.name, t.slug, tm.role, t.status
		FROM tenant_memberships tm
		JOIN tenants t ON t.id = tm.tenant_id
		WHERE tm.user_id = $1 AND tm.status = 'active'
		ORDER BY t.name
	`, userID)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var tenants []*backendv1.TenantSummary
	for rows.Next() {
		tenant := &backendv1.TenantSummary{}
		if err := rows.Scan(&tenant.Id, &tenant.Name, &tenant.Slug, &tenant.Role, &tenant.Status); err != nil {
			return nil, internal(err)
		}
		tenants = append(tenants, tenant)
	}
	if rows.Err() != nil {
		return nil, internal(rows.Err())
	}
	return tenants, nil
}

func (s *Service) productAccess(ctx context.Context, userID, tenantID string) ([]*backendv1.ProductAccess, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			p.key,
			p.name,
			COALESCE(lp.status, 'disabled') AS license_status,
			COALESCE(lp.seats_total, 0) AS seats_total,
			(
				SELECT COUNT(*)::int
				FROM product_assignments pa_count
				WHERE pa_count.tenant_id = $2 AND pa_count.product_key = p.key AND pa_count.status = 'active'
			) AS seats_used,
			COALESCE(pa.status, '') AS assignment_status,
			COALESCE(pa.role, '') AS assignment_role
		FROM products p
		LEFT JOIN product_license_pools lp ON lp.tenant_id = $2 AND lp.product_key = p.key
		LEFT JOIN product_assignments pa ON pa.tenant_id = $2 AND pa.user_id = $1 AND pa.product_key = p.key
		ORDER BY p.key
	`, userID, tenantID)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var products []*backendv1.ProductAccess
	for rows.Next() {
		product := &backendv1.ProductAccess{}
		var assignmentStatus, assignmentRole string
		if err := rows.Scan(&product.Key, &product.Name, &product.Status, &product.SeatsTotal, &product.SeatsUsed, &assignmentStatus, &assignmentRole); err != nil {
			return nil, internal(err)
		}
		product.Assigned = assignmentStatus == "active" && (product.Status == "active" || product.Status == "trial")
		if product.Assigned {
			product.Role = assignmentRole
			product.Permissions = permissionStrings(s.accessPolicy.PermissionsForProductRole(entitlements.ProductKey(product.Key), entitlements.ProductRole(assignmentRole)))
		}
		products = append(products, product)
	}
	if rows.Err() != nil {
		return nil, internal(rows.Err())
	}
	return products, nil
}

func (s *Service) requireTenantRole(ctx context.Context, session *SessionRow, allowed ...entitlements.TenantRole) error {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT tm.role
		FROM tenant_memberships tm
		JOIN tenants t ON t.id = tm.tenant_id
		WHERE tm.tenant_id = $1 AND tm.user_id = $2 AND tm.status = 'active' AND t.status = 'active'
	`, session.ActiveTenantID, session.UserID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.New(apperrors.CodeTenantAccessDenied)
		}
		return internal(err)
	}
	if !s.accessPolicy.TenantRoleAllows(entitlements.TenantRole(role), allowed...) {
		return apperrors.New(apperrors.CodePermissionDenied)
	}
	return nil
}

func permissionStrings(permissions []entitlements.Permission) []string {
	values := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		values = append(values, string(permission))
	}
	return values
}
