package main

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/Uranury/oci-platform/internal/auth"
	"github.com/Uranury/oci-platform/internal/config"
	"github.com/Uranury/oci-platform/internal/db"
	"github.com/Uranury/oci-platform/internal/handler"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/seed"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/Uranury/oci-platform/web"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

func main() {
	if err := run(); err != nil {
		slog.Error("exiting", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel})))
	slog.Debug("config loaded",
		"listen", net.JoinHostPort(cfg.ListenHost, cfg.Port),
		"log_level", cfg.LogLevel,
		"access_token_ttl", cfg.AccessTokenTTL,
		"refresh_token_ttl", cfg.RefreshTokenTTL,
	)

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	if cfg.LogLevel <= slog.LevelDebug {
		// Every query with its args and duration. Debug only: args include
		// emails and password/token hashes (never raw tokens), fine for a
		// local DB but not for production logs.
		poolCfg.ConnConfig.Tracer = &tracelog.TraceLog{
			Logger: tracelog.LoggerFunc(func(ctx context.Context, _ tracelog.LogLevel, msg string, data map[string]any) {
				if msg == "Prepare" && data["err"] == nil {
					return // the Query line right after carries the same SQL
				}
				attrs := make([]any, 0, 2*len(data))
				for _, k := range slices.Sorted(maps.Keys(data)) { // stable field order
					attrs = append(attrs, k, data[k])
				}
				slog.DebugContext(ctx, "pgx "+msg, attrs...)
			}),
			LogLevel: tracelog.LogLevelInfo, // queries and errors, not pool acquire/release noise
		}
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	userRepo := repo.NewUser(pool)
	refreshTokenRepo := repo.NewRefreshToken(pool)
	exerciseRepo := repo.NewExercise(pool)
	workoutRepo := repo.NewWorkout(pool)
	workoutSetRepo := repo.NewWorkoutSet(pool)
	templateRepo := repo.NewTemplate(pool)
	txProvider := dbutil.NewTxProvider(pool)

	// After migrations, so the library columns exist.
	if err := seed.Sync(ctx, txProvider); err != nil {
		return err
	}

	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.AccessTokenTTL)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, txProvider, jwtIssuer, cfg.RefreshTokenTTL)
	exerciseService := service.NewExerciseService(exerciseRepo)
	workoutService := service.NewWorkoutService(exerciseRepo, workoutRepo, workoutSetRepo)
	templateService := service.NewTemplateService(txProvider, templateRepo, exerciseRepo, workoutRepo, workoutSetRepo)

	authHandler := handler.NewAuthHandler(authService)
	exerciseHandler := handler.NewExerciseHandler(exerciseService)
	workoutHandler := handler.NewWorkoutHandler(workoutService)
	templateHandler := handler.NewTemplateHandler(templateService)

	webFS, err := web.Static()
	if err != nil {
		return err
	}

	router := handler.NewRouter(authHandler, exerciseHandler, workoutHandler, templateHandler, jwtIssuer, webFS)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.ListenHost, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
