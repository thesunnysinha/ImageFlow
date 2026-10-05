package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"app/internal/config"
	"app/internal/database"
	"app/internal/httpapi"
	"app/internal/jobs"
	"app/internal/safeurl"
	"app/internal/storage"
	"app/internal/worker"
	"app/migrations"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe /api/v1/health and exit (used by the container HEALTHCHECK)")
	flag.Parse()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if *healthcheck {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8000"
		}
		os.Exit(probe(port))
	}
	cfg, err := config.Load(nil)
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	if err := run(log, cfg); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// pgxpool connects lazily, so the server can start (and answer the liveness probe) before Postgres is up.
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := jobs.NewPGStore(pool)
	var files storage.Storage
	switch cfg.StorageBackend {
	case "s3":
		s3, err := storage.NewS3(ctx, storage.S3Config{
			Endpoint: cfg.S3.Endpoint, Bucket: cfg.S3.Bucket, Region: cfg.S3.Region, AccessKey: cfg.S3.AccessKey,
			SecretKey: cfg.S3.SecretKey, UseSSL: cfg.S3.UseSSL, Prefix: cfg.S3.Prefix,
		})
		if err != nil {
			return err
		}
		files = s3
	default:
		local, err := storage.NewLocal(cfg.StorageDir)
		if err != nil {
			return err
		}
		files = local
	}

	srv := &http.Server{
		Addr: cfg.Addr(),
		Handler: httpapi.New(httpapi.Dependencies{
			Store:          store,
			Storage:        files,
			APIKeys:        cfg.APIKeys,
			MaxItemsPerJob: cfg.MaxItemsPerJob,
			ValidateURL:    func(ctx context.Context, u string) error { return safeurl.Validate(ctx, u, nil) },
			Logger:         log,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", cfg.Addr())

	// Apply migrations once the database is reachable; /api/v1/ready reports the database meanwhile.
	go func() {
		for ctx.Err() == nil {
			err := database.Migrate(ctx, pool, migrations.FS)
			if err == nil {
				log.Info("migrations applied")
				return
			}
			log.Warn("could not prepare the database, retrying", "err", err)
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
			}
		}
	}()

	// The worker shares the process (set RUN_WORKER=false on API-only replicas); the queue is
	// database-backed, so any number of workers can run side by side.
	workerDone := make(chan struct{})
	if cfg.RunWorker {
		wcfg := worker.DefaultConfig()
		wcfg.Concurrency = cfg.WorkerCount
		wcfg.WebhookSecret = cfg.WebhookSecret
		w := &worker.Worker{
			Queue: store, Storage: files, Config: wcfg, Log: log,
			Fetch: safeurl.NewClient(60*time.Second, false, true),
			Hooks: safeurl.NewClient(15*time.Second, false, false),
		}
		go func() { defer close(workerDone); w.Run(ctx) }()
	} else {
		close(workerDone)
	}

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := srv.Shutdown(shutdown)
		select { // let in-flight images finish recording their result
		case <-workerDone:
		case <-shutdown.Done():
		}
		return err
	}
	return nil
}

func probe(port string) int {
	client := http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort("127.0.0.1", port) + "/api/v1/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
