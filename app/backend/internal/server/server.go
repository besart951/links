package server

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/links/backend/internal/config"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	"github.com/links/backend/internal/gen/proto/links/backend/v1/backendv1connect"
	"github.com/links/backend/internal/platform"
	entitlements "github.com/links/backend/internal/platform/entitlements/domain"
)

type backendRPCServer struct {
	pool     *pgxpool.Pool
	cfg      config.Config
	platform *platform.Service
}

// New returns the root HTTP handler with all routes registered.
func New(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	svc := &backendRPCServer{
		pool:     pool,
		cfg:      cfg,
		platform: platform.New(pool, cfg),
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if pool != nil {
			if err := pool.Ping(r.Context()); err != nil {
				dbStatus = "unreachable"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"status\":\"ok\",\"db\":\"" + dbStatus + "\"}"))
	})

	register := func(path string, handler http.Handler) {
		mux.Handle(path, withCORS(cfg, handler))
	}
	register(backendv1connect.NewBackendServiceHandler(svc))
	register(backendv1connect.NewAuthServiceHandler(svc))
	register(backendv1connect.NewTenantServiceHandler(svc))
	register(backendv1connect.NewEntitlementServiceHandler(svc))
	register(backendv1connect.NewMembershipServiceHandler(svc))
	register(backendv1connect.NewLicensingServiceHandler(svc))
	register(backendv1connect.NewAssignmentServiceHandler(svc))
	register(backendv1connect.NewDemoServiceHandler(svc))

	return mux
}

func (s *backendRPCServer) GetHealth(ctx context.Context, _ *connect.Request[backendv1.GetHealthRequest]) (*connect.Response[backendv1.GetHealthResponse], error) {
	dbStatus := "ok"
	if s.pool != nil {
		if err := s.pool.Ping(ctx); err != nil {
			dbStatus = "unreachable"
		}
	}
	return connect.NewResponse(&backendv1.GetHealthResponse{
		Status: "ok",
		Db:     dbStatus,
	}), nil
}

func (s *backendRPCServer) Register(ctx context.Context, req *connect.Request[backendv1.RegisterRequest]) (*connect.Response[backendv1.AuthResponse], error) {
	auth, token, err := s.platform.Register(ctx, req.Msg.Email, req.Msg.Password, req.Msg.DisplayName, req.Msg.TenantName, req.Header().Get("User-Agent"))
	if err != nil {
		return nil, rpcError(err)
	}
	res := connect.NewResponse(auth)
	res.Header().Add("Set-Cookie", s.platform.SessionCookie(token).String())
	return res, nil
}

func (s *backendRPCServer) Login(ctx context.Context, req *connect.Request[backendv1.LoginRequest]) (*connect.Response[backendv1.AuthResponse], error) {
	auth, token, err := s.platform.Login(ctx, req.Msg.Email, req.Msg.Password, req.Header().Get("User-Agent"))
	if err != nil {
		return nil, rpcError(err)
	}
	res := connect.NewResponse(auth)
	res.Header().Add("Set-Cookie", s.platform.SessionCookie(token).String())
	return res, nil
}

func (s *backendRPCServer) Logout(ctx context.Context, req *connect.Request[backendv1.LogoutRequest]) (*connect.Response[backendv1.LogoutResponse], error) {
	if err := s.platform.Logout(ctx, tokenFromHeader(req.Header(), s.platform.CookieName())); err != nil {
		return nil, rpcError(err)
	}
	res := connect.NewResponse(&backendv1.LogoutResponse{Ok: true})
	res.Header().Add("Set-Cookie", s.platform.ExpiredSessionCookie().String())
	return res, nil
}

func (s *backendRPCServer) GetSession(ctx context.Context, req *connect.Request[backendv1.GetSessionRequest]) (*connect.Response[backendv1.GetSessionResponse], error) {
	session, err := s.platform.GetSessionResponse(ctx, tokenFromHeader(req.Header(), s.platform.CookieName()))
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(session), nil
}

func (s *backendRPCServer) ListMyTenants(ctx context.Context, req *connect.Request[backendv1.ListMyTenantsRequest]) (*connect.Response[backendv1.ListMyTenantsResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.ListMyTenants(ctx, session)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) SwitchTenant(ctx context.Context, req *connect.Request[backendv1.SwitchTenantRequest]) (*connect.Response[backendv1.AccessSnapshotResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.SwitchTenant(ctx, session, req.Msg.TenantId)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) GetAccessSnapshot(ctx context.Context, req *connect.Request[backendv1.GetAccessSnapshotRequest]) (*connect.Response[backendv1.AccessSnapshotResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	access, err := s.platform.BuildAccessSnapshot(ctx, session.UserID, session.ActiveTenantID)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&backendv1.AccessSnapshotResponse{Access: access}), nil
}

func (s *backendRPCServer) ListTenantMembers(ctx context.Context, req *connect.Request[backendv1.ListTenantMembersRequest]) (*connect.Response[backendv1.ListTenantMembersResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.ListTenantMembers(ctx, session)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) InviteTenantMember(ctx context.Context, req *connect.Request[backendv1.InviteTenantMemberRequest]) (*connect.Response[backendv1.InviteTenantMemberResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.InviteTenantMember(ctx, session, req.Msg.Email, req.Msg.DisplayName, req.Msg.TenantRole)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) AcceptTenantInvite(ctx context.Context, req *connect.Request[backendv1.AcceptTenantInviteRequest]) (*connect.Response[backendv1.AuthResponse], error) {
	auth, token, err := s.platform.AcceptInvite(ctx, req.Msg.InviteToken, req.Msg.DisplayName, req.Msg.Password, req.Header().Get("User-Agent"))
	if err != nil {
		return nil, rpcError(err)
	}
	res := connect.NewResponse(auth)
	res.Header().Add("Set-Cookie", s.platform.SessionCookie(token).String())
	return res, nil
}

func (s *backendRPCServer) GetTenantLicenses(ctx context.Context, req *connect.Request[backendv1.GetTenantLicensesRequest]) (*connect.Response[backendv1.GetTenantLicensesResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.GetTenantLicenses(ctx, session)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) SetMyTenantProductSeats(ctx context.Context, req *connect.Request[backendv1.SetMyTenantProductSeatsRequest]) (*connect.Response[backendv1.SetMyTenantProductSeatsResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.SetMyTenantProductSeats(ctx, session, req.Msg.ProductKey, req.Msg.SeatsTotal)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) ListLicenseEvents(ctx context.Context, req *connect.Request[backendv1.ListLicenseEventsRequest]) (*connect.Response[backendv1.ListLicenseEventsResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.ListLicenseEvents(ctx, session)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) ListProductAssignments(ctx context.Context, req *connect.Request[backendv1.ListProductAssignmentsRequest]) (*connect.Response[backendv1.ListProductAssignmentsResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.ListProductAssignments(ctx, session, req.Msg.ProductKey)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) AssignUserToProduct(ctx context.Context, req *connect.Request[backendv1.AssignUserToProductRequest]) (*connect.Response[backendv1.ProductAssignment], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.AssignUserToProduct(ctx, session, req.Msg.UserId, req.Msg.ProductKey, req.Msg.Role)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) RemoveUserFromProduct(ctx context.Context, req *connect.Request[backendv1.RemoveUserFromProductRequest]) (*connect.Response[backendv1.ProductAssignment], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.RemoveUserFromProduct(ctx, session, req.Msg.UserId, req.Msg.ProductKey)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) UpdateProductRole(ctx context.Context, req *connect.Request[backendv1.UpdateProductRoleRequest]) (*connect.Response[backendv1.ProductAssignment], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	res, err := s.platform.UpdateProductRole(ctx, session, req.Msg.UserId, req.Msg.ProductKey, req.Msg.Role)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(res), nil
}

func (s *backendRPCServer) GetPlannerDemo(ctx context.Context, req *connect.Request[backendv1.GetPlannerDemoRequest]) (*connect.Response[backendv1.DemoFeatureResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	if err := s.platform.RequireProductPermission(ctx, session, entitlements.ProductPlanner, entitlements.PermissionPlannerTaskRead); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&backendv1.DemoFeatureResponse{
		ProductKey: string(entitlements.ProductPlanner),
		Title:      "PlannerLink Demo",
		Message:    "Du hast Zugriff auf PlannerLink.",
	}), nil
}

func (s *backendRPCServer) GetFinanceDemo(ctx context.Context, req *connect.Request[backendv1.GetFinanceDemoRequest]) (*connect.Response[backendv1.DemoFeatureResponse], error) {
	session, err := s.session(ctx, req.Header())
	if err != nil {
		return nil, rpcError(err)
	}
	if err := s.platform.RequireProductPermission(ctx, session, entitlements.ProductFinance, entitlements.PermissionFinanceInvoiceRead); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&backendv1.DemoFeatureResponse{
		ProductKey: string(entitlements.ProductFinance),
		Title:      "FinanceLink Demo",
		Message:    "Du hast Zugriff auf FinanceLink.",
	}), nil
}

func (s *backendRPCServer) session(ctx context.Context, header http.Header) (*platform.SessionRow, error) {
	return s.platform.SessionFromToken(ctx, tokenFromHeader(header, s.platform.CookieName()))
}

func tokenFromHeader(header http.Header, cookieName string) string {
	auth := header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	req := &http.Request{Header: header}
	cookie, err := req.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func withCORS(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(cfg.AllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if cfg.AllowedOrigins == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST,GET,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Connect-Protocol-Version,Connect-Timeout-Ms,Grpc-Timeout,X-Grpc-Web,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(allowed, origin string) bool {
	if allowed == "*" || allowed == "" {
		return true
	}
	for _, candidate := range strings.Split(allowed, ",") {
		if strings.TrimSpace(candidate) == origin {
			return true
		}
	}
	return false
}
