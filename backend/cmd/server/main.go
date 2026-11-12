package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/istiyak/secure-exam-eval/internal/api"
	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/config"
	"github.com/istiyak/secure-exam-eval/internal/db"
	"github.com/istiyak/secure-exam-eval/internal/storage"
	"github.com/istiyak/secure-exam-eval/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	level := slog.LevelInfo
	if cfg.Env == "development" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	files, err := storage.New(cfg.StorageDriver, cfg.StorageDir, cfg.S3Bucket, cfg.S3Region, cfg.S3Endpoint)
	if err != nil {
		return err
	}

	st := store.New(pool)
	if err := bootstrapAdmin(ctx, st, cfg, logger); err != nil {
		return err
	}

	srv := &api.Server{
		Cfg:     cfg,
		DB:      st,
		Tokens:  auth.NewManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL),
		Files:   files,
		Log:     logger,
		Limiter: api.NewLimiter(10, time.Minute),
	}

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "env", cfg.Env, "storage", cfg.StorageDriver)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
}

// bootstrapAdmin seeds the first administrator once, from env.
func bootstrapAdmin(ctx context.Context, st *store.Store, cfg *config.Config, log *slog.Logger) error {
	if cfg.BootstrapEmail == "" || cfg.BootstrapPass == "" {
		return nil
	}
	if _, err := st.UserByEmail(ctx, cfg.BootstrapEmail); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	hash, err := auth.HashPassword(cfg.BootstrapPass)
	if err != nil {
		return err
	}
	u, err := st.CreateUser(ctx, cfg.BootstrapEmail, hash, "System Administrator", string(auth.RoleAdmin))
	if err != nil {
		return err
	}
	log.Info("bootstrap admin created", "email", u.Email, "id", u.ID)
	return nil
}
