package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/guilhermelinosp/hellnet-lib-api/config"
)

type Server struct {
	http            *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger, handler http.Handler) *Server {
	if logger == nil { logger = slog.Default() }
	return &Server{
		http: &http.Server{
			Addr: ":" + cfg.Port, Handler: handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout, ReadTimeout: cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout,
			MaxHeaderBytes: 1 << 20,
		},
		shutdownTimeout: cfg.ShutdownTimeout, logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	serveErr := make(chan error, 1)
	go func() {
		s.logger.Info("HTTP server listening", slog.String("addr", s.http.Addr), slog.String("read_timeout", s.http.ReadTimeout.String()))
		serveErr <- s.http.ListenAndServe()
	}()
	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) { return nil }
		return fmt.Errorf("server: listenAndServe failed: %w", err)
	case <-ctx.Done():
		drainCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.http.Shutdown(drainCtx); err != nil { return fmt.Errorf("server: graceful shutdown incomplete: %w", err) }
		s.logger.Info("HTTP server drained connections gracefully")
		return nil
	}
}
