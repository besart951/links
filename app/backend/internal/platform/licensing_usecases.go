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

func (s *Service) GetTenantLicenses(ctx context.Context, session *SessionRow) (*backendv1.GetTenantLicensesResponse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			COALESCE(lp.id::text, ''),
			p.key,
			p.name,
			COALESCE(lp.seats_total, 0),
			(
				SELECT COUNT(*)::int
				FROM product_assignments pa
				WHERE pa.tenant_id = $1 AND pa.product_key = p.key AND pa.status = 'active'
			) AS seats_used,
			COALESCE(lp.status, 'disabled'),
			COALESCE(lp.source, ''),
			COALESCE(lp.updated_at, p.created_at)
		FROM products p
		LEFT JOIN product_license_pools lp ON lp.product_key = p.key AND lp.tenant_id = $1
		ORDER BY p.key
	`, session.ActiveTenantID)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var licenses []*backendv1.ProductLicense
	for rows.Next() {
		license := &backendv1.ProductLicense{}
		var updatedAt time.Time
		if err := rows.Scan(&license.Id, &license.ProductKey, &license.ProductName, &license.SeatsTotal, &license.SeatsUsed, &license.Status, &license.Source, &updatedAt); err != nil {
			return nil, internal(err)
		}
		license.UpdatedAt = updatedAt.Format(time.RFC3339)
		licenses = append(licenses, license)
	}
	if rows.Err() != nil {
		return nil, internal(rows.Err())
	}
	return &backendv1.GetTenantLicensesResponse{Licenses: licenses}, nil
}

func (s *Service) SetMyTenantProductSeats(ctx context.Context, session *SessionRow, productKey string, seatsTotal int32) (*backendv1.SetMyTenantProductSeatsResponse, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleBillingAdmin); err != nil {
		return nil, err
	}
	if err := s.licensePolicy.ValidateSelfServiceSeats(seatsTotal, 0); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, internal(err)
	}
	defer rollback(ctx, tx)

	if err := requireProductExists(ctx, tx, productKey); err != nil {
		return nil, err
	}
	var seatsUsed int32
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM product_assignments
		WHERE tenant_id = $1 AND product_key = $2 AND status = 'active'
	`, session.ActiveTenantID, productKey).Scan(&seatsUsed); err != nil {
		return nil, internal(err)
	}
	if err := s.licensePolicy.ValidateSelfServiceSeats(seatsTotal, seatsUsed); err != nil {
		return nil, err
	}

	var existingID sql.NullString
	var before int32
	err = tx.QueryRow(ctx, `
		SELECT id::text, seats_total
		FROM product_license_pools
		WHERE tenant_id = $1 AND product_key = $2
		FOR UPDATE
	`, session.ActiveTenantID, productKey).Scan(&existingID, &before)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, internal(err)
	}
	eventType := s.licensePolicy.ChangeType(existingID.Valid, before, seatsTotal)
	if existingID.Valid {
		_, err = tx.Exec(ctx, `
			UPDATE product_license_pools
			SET seats_total = $1, status = 'active', source = 'manual_self_service', updated_at = now(), created_by = $2
			WHERE id = $3
		`, seatsTotal, session.UserID, existingID.String)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_license_pools (tenant_id, product_key, seats_total, status, source, created_by)
			VALUES ($1, $2, $3, 'active', 'manual_self_service', $4)
		`, session.ActiveTenantID, productKey, seatsTotal, session.UserID)
	}
	if err != nil {
		return nil, internal(err)
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO license_events (tenant_id, product_key, type, seats_before, seats_after, source, actor_user_id)
		VALUES ($1, $2, $3, $4, $5, 'manual_self_service', $6)
	`, session.ActiveTenantID, productKey, eventType, before, seatsTotal, session.UserID)
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, tenant_id, action, target_type, target_id)
		VALUES ($1, $2, 'license.pool.updated', 'product', $3)
	`, session.UserID, session.ActiveTenantID, productKey)
	if err := tx.Commit(ctx); err != nil {
		return nil, internal(err)
	}
	licenses, err := s.GetTenantLicenses(ctx, session)
	if err != nil {
		return nil, err
	}
	for _, license := range licenses.Licenses {
		if license.ProductKey == productKey {
			return &backendv1.SetMyTenantProductSeatsResponse{License: license}, nil
		}
	}
	return nil, apperrors.New(apperrors.CodeNotFound)
}

func (s *Service) ListLicenseEvents(ctx context.Context, session *SessionRow) (*backendv1.ListLicenseEventsResponse, error) {
	if err := s.requireTenantRole(ctx, session, entitlements.TenantRoleBillingAdmin); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, product_key, type, seats_before, seats_after, source, COALESCE(actor_user_id::text, ''), created_at
		FROM license_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, session.ActiveTenantID)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()
	var events []*backendv1.LicenseEvent
	for rows.Next() {
		event := &backendv1.LicenseEvent{}
		var createdAt time.Time
		if err := rows.Scan(&event.Id, &event.ProductKey, &event.Type, &event.SeatsBefore, &event.SeatsAfter, &event.Source, &event.ActorUserId, &createdAt); err != nil {
			return nil, internal(err)
		}
		event.CreatedAt = createdAt.Format(time.RFC3339)
		events = append(events, event)
	}
	if rows.Err() != nil {
		return nil, internal(rows.Err())
	}
	return &backendv1.ListLicenseEventsResponse{Events: events}, nil
}

func requireProductExists(ctx context.Context, tx pgx.Tx, productKey string) error {
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE key = $1 AND status = 'active')`, productKey).Scan(&ok); err != nil {
		return internal(err)
	}
	if !ok {
		return apperrors.New(apperrors.CodeProductNotFound)
	}
	return nil
}
