package platform

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	entitlements "github.com/links/backend/internal/platform/entitlements/domain"
)

func (s *Service) ListTenantMembers(ctx context.Context, session *SessionRow) (*backendv1.ListTenantMembersResponse, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleAdmin); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.email, u.display_name, tm.role, tm.status
		FROM tenant_memberships tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.tenant_id = $1 AND tm.status <> 'removed'
		ORDER BY u.email
	`, session.ActiveTenantID)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var members []*backendv1.TenantMember
	for rows.Next() {
		member := &backendv1.TenantMember{}
		if err := rows.Scan(&member.UserId, &member.Email, &member.DisplayName, &member.TenantRole, &member.Status); err != nil {
			return nil, internal(err)
		}
		members = append(members, member)
	}
	return &backendv1.ListTenantMembersResponse{Members: members}, rows.Err()
}

func (s *Service) InviteTenantMember(ctx context.Context, session *SessionRow, email, displayName, tenantRole string) (*backendv1.InviteTenantMemberResponse, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleAdmin); err != nil {
		return nil, err
	}
	email = normalizeEmail(email)
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = email
	}
	if tenantRole == "" {
		tenantRole = string(entitlements.TenantRoleMember)
	}
	if !s.accessPolicy.ValidTenantRole(entitlements.TenantRole(tenantRole)) || entitlements.TenantRole(tenantRole) == entitlements.TenantRoleOwner {
		return nil, invalid("INVALID_TENANT_ROLE")
	}
	inviteToken, err := randomToken()
	if err != nil {
		return nil, internal(err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, internal(err)
	}
	defer rollback(ctx, tx)

	userID, err := upsertInvitedUser(ctx, tx, email, displayName)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_memberships (tenant_id, user_id, role, status)
		VALUES ($1, $2, $3, 'invited')
		ON CONFLICT (tenant_id, user_id)
		DO UPDATE SET role = EXCLUDED.role, status = CASE WHEN tenant_memberships.status = 'removed' THEN 'invited' ELSE tenant_memberships.status END, updated_at = now()
	`, session.ActiveTenantID, userID, tenantRole); err != nil {
		return nil, internal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_invites (tenant_id, user_id, email, tenant_role, token_hash, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ActiveTenantID, userID, email, tenantRole, hashToken(inviteToken), time.Now().Add(InviteDuration), session.UserID); err != nil {
		return nil, internal(err)
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'tenant.member.invited', 'user', $3)
	`, session.UserID, session.ActiveTenantID, userID)
	if err := tx.Commit(ctx); err != nil {
		return nil, internal(err)
	}
	return &backendv1.InviteTenantMemberResponse{
		Member: &backendv1.TenantMember{
			UserId:      userID,
			Email:       email,
			DisplayName: displayName,
			TenantRole:  tenantRole,
			Status:      "invited",
		},
		InviteToken: inviteToken,
	}, nil
}

func upsertInvitedUser(ctx context.Context, tx pgx.Tx, email, displayName string) (string, error) {
	var userID string
	err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE email = $1`, email).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", internal(err)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, status, platform_role)
		VALUES ($1, $2, 'invited', 'none')
		RETURNING id::text
	`, email, displayName).Scan(&userID)
	if err != nil {
		return "", internal(err)
	}
	return userID, nil
}
