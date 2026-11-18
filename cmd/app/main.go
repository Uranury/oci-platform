package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
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
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
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
		log.Printf("listening on :%s", cfg.Port)
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
		log.Println("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
