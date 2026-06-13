package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	apperrors "github.com/links/backend/pkg/shared/errors"
)

func (s *Service) SessionFromToken(ctx context.Context, token string) (*SessionRow, error) {
	if token == "" {
		return nil, apperrors.New(apperrors.CodeUnauthenticated)
	}
	var session SessionRow
	err := s.pool.QueryRow(ctx, `
		SELECT s.id::text, s.user_id::text, s.active_tenant_id::text, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		  AND u.status = 'active'
	`, hashToken(token)).Scan(&session.ID, &session.UserID, &session.ActiveTenantID, &session.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.CodeSessionExpired)
		}
		return nil, internal(err)
	}
	return &session, nil
}

func (s *Service) SessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionDuration),
		MaxAge:   int(SessionDuration.Seconds()),
	}
}

func (s *Service) ExpiredSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func (s *Service) CookieName() string {
	return s.cfg.SessionCookieName
}

func (s *Service) createSessionTx(ctx context.Context, tx pgx.Tx, userID, tenantID, sessionType, userAgent string) (*SessionRow, string, error) {
	token, err := randomToken()
	if err != nil {
		return nil, "", err
	}
	expiresAt := time.Now().Add(SessionDuration)
	session := &SessionRow{UserID: userID, ActiveTenantID: tenantID, ExpiresAt: expiresAt}
	err = tx.QueryRow(ctx, `
		INSERT INTO sessions (user_id, active_tenant_id, type, token_hash, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, userID, tenantID, sessionType, hashToken(token), userAgent, expiresAt).Scan(&session.ID)
	if err != nil {
		return nil, "", err
	}
	return session, token, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sessionProto(session *SessionRow) *backendv1.Session {
	if session == nil {
		return nil
	}
	return &backendv1.Session{
		Id:             session.ID,
		UserId:         session.UserID,
		ActiveTenantId: session.ActiveTenantID,
		ExpiresAt:      session.ExpiresAt.Format(time.RFC3339),
	}
}
