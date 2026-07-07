package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"firewall-manager/backend/internal/auth"
	"firewall-manager/backend/internal/config"
	"firewall-manager/backend/internal/firewall"
	"firewall-manager/backend/internal/httpapi"
	"firewall-manager/backend/internal/staticweb"
)

func Run(logger *slog.Logger) error {
	cfg := config.Load()
	sessions := auth.NewSessionManager([]byte(cfg.Auth.SessionSecret), cfg.Auth.SessionTTL)
	firewallService, err := firewall.NewService(context.Background(), cfg.Firewall, logger)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	httpapi.Register(mux, httpapi.Dependencies{
		Config:          cfg,
		Logger:          logger,
		Sessions:        sessions,
		FirewallService: firewallService,
	})
	staticweb.Register(mux)

	server := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           requestLogger(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting firewall-manager", "addr", cfg.Server.Addr(), "tls", cfg.Server.TLS.Enabled)
		if cfg.Server.TLS.Enabled {
			errCh <- server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
			return
		}
		if !cfg.Server.IsLoopback() && !cfg.Server.AllowInsecureRemote {
			errCh <- fmt.Errorf("refusing insecure remote HTTP listener on %s; enable TLS or set allow_insecure_remote", cfg.Server.Host)
			return
		}
		if !cfg.Server.IsLoopback() {
			logger.Warn("starting without TLS on non-loopback address", "host", cfg.Server.Host)
		}
		errCh <- server.ListenAndServe()
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}
