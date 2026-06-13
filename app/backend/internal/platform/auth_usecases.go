package platform

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	apperrors "github.com/links/backend/pkg/shared/errors"
)

func (s *Service) Register(ctx context.Context, email, password, displayName, tenantName, userAgent string) (*backendv1.AuthResponse, string, error) {
	email = normalizeEmail(email)
	displayName = strings.TrimSpace(displayName)
	tenantName = strings.TrimSpace(tenantName)
	if email == "" || !strings.Contains(email, "@") {
		return nil, "", invalid("INVALID_EMAIL")
	}
	if displayName == "" {
		displayName = email
	}
	if tenantName == "" {
		tenantName = displayName + " Workspace"
	}
	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return nil, "", invalid(err.Error())
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, "", internal(err)
	}
	defer rollback(ctx, tx)

	platformRole := "none"
	if s.cfg.BootstrapSuperAdminEmail != "" && email == normalizeEmail(s.cfg.BootstrapSuperAdminEmail) {
		platformRole = "super_admin"
	}

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name, status, platform_role)
		VALUES ($1, $2, $3, 'active', $4)
		RETURNING id::text
	`, email, passwordHash, displayName, platformRole).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, "", apperrors.New(apperrors.CodeEmailAlreadyExists)
		}
		return nil, "", internal(err)
	}

	slug, err := uniqueTenantSlug(ctx, tx, tenantName)
	if err != nil {
		return nil, "", internal(err)
	}
	var tenantID string
	err = tx.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, status)
		VALUES ($1, $2, 'active')
		RETURNING id::text
	`, tenantName, slug).Scan(&tenantID)
	if err != nil {
		return nil, "", internal(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_memberships (tenant_id, user_id, role, status)
		VALUES ($1, $2, 'owner', 'active')
	`, tenantID, userID); err != nil {
		return nil, "", internal(err)
	}

	session, token, err := s.createSessionTx(ctx, tx, userID, tenantID, "web_cookie", userAgent)
	if err != nil {
		return nil, "", internal(err)
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'user.registered', 'user', $1)
	`, userID, tenantID)

	if err := tx.Commit(ctx); err != nil {
		return nil, "", internal(err)
	}

	access, err := s.BuildAccessSnapshot(ctx, userID, tenantID)
	if err != nil {
		return nil, "", err
	}
	return &backendv1.AuthResponse{Session: sessionProto(session), Access: access}, token, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent string) (*backendv1.AuthResponse, string, error) {
	email = normalizeEmail(email)
	var userID, passwordHash, status string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(password_hash, ''), status
		FROM users
		WHERE email = $1
	`, email).Scan(&userID, &passwordHash, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", apperrors.New(apperrors.CodeInvalidCredentials)
		}
		return nil, "", internal(err)
	}
	if status != "active" || passwordHash == "" {
		return nil, "", apperrors.New(apperrors.CodeUserNotActive)
	}
	ok, err := s.passwordHasher.Verify(password, passwordHash)
	if err != nil || !ok {
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO audit_events (actor_user_id, action, target_type, target_id)
			VALUES ($1, 'user.login.failed', 'user', $1)
		`, userID)
		return nil, "", apperrors.New(apperrors.CodeInvalidCredentials)
	}

	activeTenantID, err := s.firstActiveTenantID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, "", internal(err)
	}
	defer rollback(ctx, tx)
	session, token, err := s.createSessionTx(ctx, tx, userID, activeTenantID, "web_cookie", userAgent)
	if err != nil {
		return nil, "", internal(err)
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'user.login.success', 'user', $1)
	`, userID, activeTenantID)
	if err := tx.Commit(ctx); err != nil {
		return nil, "", internal(err)
	}
	access, err := s.BuildAccessSnapshot(ctx, userID, activeTenantID)
	if err != nil {
		return nil, "", err
	}
	return &backendv1.AuthResponse{Session: sessionProto(session), Access: access}, token, nil
}

func (s *Service) AcceptInvite(ctx context.Context, inviteToken, displayName, password, userAgent string) (*backendv1.AuthResponse, string, error) {
	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return nil, "", invalid(err.Error())
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, "", internal(err)
	}
	defer rollback(ctx, tx)

	var inviteID, userID, tenantID string
	var email, currentName string
	err = tx.QueryRow(ctx, `
		SELECT ti.id::text, ti.user_id::text, ti.tenant_id::text, u.email, u.display_name
		FROM tenant_invites ti
		JOIN users u ON u.id = ti.user_id
		WHERE ti.token_hash = $1 AND ti.status = 'pending' AND ti.expires_at > now()
	`, hashToken(inviteToken)).Scan(&inviteID, &userID, &tenantID, &email, &currentName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", apperrors.New(apperrors.CodeInviteNotFound)
		}
		return nil, "", internal(err)
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = currentName
	}
	if displayName == "" {
		displayName = email
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $1, display_name = $2, status = 'active', updated_at = now()
		WHERE id = $3
	`, passwordHash, displayName, userID); err != nil {
		return nil, "", internal(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tenant_memberships SET status = 'active', updated_at = now()
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID); err != nil {
		return nil, "", internal(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tenant_invites SET status = 'accepted', accepted_at = now()
		WHERE id = $1
	`, inviteID); err != nil {
		return nil, "", internal(err)
	}
	session, token, err := s.createSessionTx(ctx, tx, userID, tenantID, "web_cookie", userAgent)
	if err != nil {
		return nil, "", internal(err)
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'tenant.invite.accepted', 'tenant_invite', $3)
	`, userID, tenantID, inviteID)
	if err := tx.Commit(ctx); err != nil {
		return nil, "", internal(err)
	}
	access, err := s.BuildAccessSnapshot(ctx, userID, tenantID)
	if err != nil {
		return nil, "", err
	}
	return &backendv1.AuthResponse{Session: sessionProto(session), Access: access}, token, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, hashToken(token))
	if err != nil {
		return internal(err)
	}
	return nil
}

func (s *Service) GetSessionResponse(ctx context.Context, token string) (*backendv1.GetSessionResponse, error) {
	session, err := s.SessionFromToken(ctx, token)
	if err != nil {
		if code, ok := apperrors.CodeOf(err); ok && (code == apperrors.CodeUnauthenticated || code == apperrors.CodeSessionExpired) {
			return &backendv1.GetSessionResponse{Authenticated: false}, nil
		}
		return nil, err
	}
	access, err := s.BuildAccessSnapshot(ctx, session.UserID, session.ActiveTenantID)
	if err != nil {
		return nil, err
	}
	return &backendv1.GetSessionResponse{
		Authenticated: true,
		Session:       sessionProto(session),
		Access:        access,
	}, nil
}
