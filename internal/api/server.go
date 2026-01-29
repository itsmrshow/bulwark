package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
	"golang.org/x/time/rate"
)

// Server provides the HTTP API and UI handlers.
type Server struct {
	cfg          Config
	logger       *logging.Logger
	store        state.Store
	runs         *RunManager
	writeLimiter *rate.Limiter
	planCache    *planCache
}

// NewServer constructs a new API server.
func NewServer(cfg Config, logger *logging.Logger) (*Server, error) {
	if logger == nil {
		logger = logging.Default()
	}

	var store state.Store
	if cfg.StateDB != "" {
		sqliteStore, err := state.NewSQLiteStore(cfg.StateDB, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create state store: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := sqliteStore.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize state store: %w", err)
		}
		store = sqliteStore
		logger.Info().Str("path", cfg.StateDB).Msg("State persistence enabled")
	}

	var limiter *rate.Limiter
	if cfg.WriteRateRPS > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.WriteRateRPS), cfg.WriteRateBurst)
	}

	return &Server{
		cfg:          cfg,
		logger:       logger.WithComponent("api"),
		store:        store,
		runs:         NewRunManager(25, 1500, 200),
		writeLimiter: limiter,
		planCache:    newPlanCache(cfg.PlanCacheTTL),
	}, nil
}

// Close releases server resources.
func (s *Server) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// Handler returns the http.Handler with routes configured.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/overview", s.handleOverview)
	mux.HandleFunc("/api/targets", s.handleTargets)
	mux.HandleFunc("/api/targets/", s.handleTargetByID)
	mux.HandleFunc("/api/plan", s.handlePlan)
	mux.Handle("/api/apply", s.requireWrite(http.HandlerFunc(s.handleApply)))
	mux.HandleFunc("/api/runs/", s.handleRun)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.Handle("/api/rollback", s.requireWrite(http.HandlerFunc(s.handleRollback)))

	if s.cfg.UIEnabled {
		mux.Handle("/", s.uiHandler())
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	return loggingMiddleware(mux, s.logger)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	if s.cfg.UIEnabled {
		if _, err := os.Stat(s.cfg.DistDir); err != nil {
			s.logger.Warn().Err(err).Str("dist", s.cfg.DistDir).Msg("UI dist directory not found")
		}
	}

	server := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	s.logger.Info().Str("addr", s.cfg.Addr).Msg("Bulwark web server listening")
	return server.ListenAndServe()
}
