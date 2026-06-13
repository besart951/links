package platform

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/links/backend/internal/config"
	"github.com/links/backend/internal/platform/auth/adapters/security"
	authports "github.com/links/backend/internal/platform/auth/ports"
	entitlements "github.com/links/backend/internal/platform/entitlements/domain"
	licensingdomain "github.com/links/backend/internal/platform/licensing/domain"
)

const (
	SessionDuration = 7 * 24 * time.Hour
	InviteDuration  = 14 * 24 * time.Hour
)

type Service struct {
	pool           *pgxpool.Pool
	cfg            config.Config
	passwordHasher authports.PasswordHasher
	accessPolicy   entitlements.AuthorizationPolicy
	licensePolicy  licensingdomain.LicensePolicy
}

type SessionRow struct {
	ID             string
	UserID         string
	ActiveTenantID string
	ExpiresAt      time.Time
}

func New(pool *pgxpool.Pool, cfg config.Config) *Service {
	return &Service{
		pool:           pool,
		cfg:            cfg,
		passwordHasher: security.NewArgon2idPasswordHasher(),
		accessPolicy:   entitlements.NewAuthorizationPolicy(),
		licensePolicy:  licensingdomain.NewLicensePolicy(),
	}
}
