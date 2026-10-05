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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thesunnysinha/imageflow/apps/api/internal/config"
	"github.com/thesunnysinha/imageflow/apps/api/internal/httpapi"
	"github.com/thesunnysinha/imageflow/apps/api/internal/jobs"
	"github.com/thesunnysinha/imageflow/apps/api/internal/safeurl"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	router := httpapi.NewRouter(httpapi.Deps{
		Store:          jobs.NewPGStore(pool),
		APIKeys:        cfg.APIKeys,
		MaxItemsPerJob: cfg.MaxItemsPerJob,
		ValidateURL:    func(ctx context.Context, u string) error { return safeurl.Validate(ctx, u, nil) },
		Logger:         log,
		Production:     cfg.Production,
	})
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", cfg.Addr)

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(sctx)
	}
	return nil
}
