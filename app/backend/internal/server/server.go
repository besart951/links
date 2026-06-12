package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	backendv1 "github.com/links/backend/internal/gen/proto/links/backend/v1"
	"github.com/links/backend/internal/gen/proto/links/backend/v1/backendv1connect"
	"github.com/links/backend/internal/config"
)

type backendRPCServer struct {
	pool *pgxpool.Pool
}

// New returns the root HTTP handler with all routes registered.
func New(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	_ = cfg

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

	rpcPath, rpcHandler := backendv1connect.NewBackendServiceHandler(&backendRPCServer{pool: pool})
	mux.Handle(rpcPath, withCORS(rpcHandler))

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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST,GET,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Connect-Protocol-Version,Connect-Timeout-Ms,Grpc-Timeout,X-Grpc-Web")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
