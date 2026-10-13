package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/StasanDev/avitoLabs/internal/config"
	database "github.com/StasanDev/avitoLabs/internal/database/postgres"
	"github.com/StasanDev/avitoLabs/internal/handler"
	repo "github.com/StasanDev/avitoLabs/internal/repository/postgres"
	"github.com/StasanDev/avitoLabs/internal/usecase"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	// log.Println(cfg)
	log.Printf("config is loaded")

	ctx := context.Background()
	dbConnCtx, cancel := context.WithTimeout(ctx, cfg.DB.ConnectTimeout)
	defer cancel()

	pool, err := database.NewPostgresPool(dbConnCtx, &cfg.DB)
	if err != nil {
		log.Fatalf("failed to create postgres pool: %v", err)
	}
	log.Printf("connected to db")

	tripRepository := repo.NewTripRepository(pool, cfg.DB.QueryTimeout)
	tripStatusHistoryRepository := repo.NewTripStatusRepository(pool, cfg.DB.QueryTimeout)
	idempotencyRepository := repo.NewIdempotencyRepository(pool, cfg.DB.QueryTimeout)
	txManager := repo.NewTxManager(pool, cfg.DB.QueryTimeout)
	tripService := usecase.NewTripService(tripRepository, tripStatusHistoryRepository,
		idempotencyRepository, txManager, cfg.IdempotencyTTL)
	httpHandler := handler.NewHandler(tripService, pool, cfg.DB.QueryTimeout)
	router := handler.NewRouter(httpHandler)

	server := &http.Server{
		Addr:              cfg.Http.Addr,
		Handler:           router,
		ReadTimeout:       cfg.Http.ReadTimeout,
		ReadHeaderTimeout: cfg.Http.ReadHeaderTimeout,
		WriteTimeout:      cfg.Http.WriteTimeout,
		IdleTimeout:       cfg.Http.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP server is listening on %s", cfg.Http.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(
		ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server: %v", err)
		}
	case <-signalContext.Done():
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := shutdown(shutdownCtx, server.Shutdown, pool.Close); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("shutdown timeout exceeded; forcing exit: %v", err)
			return
		}
		log.Printf("graceful shutdown failed: %v", err)
	}

	log.Printf("service stopped")
}

func shutdown(ctx context.Context, shutdownHTTP func(context.Context) error, close func()) error {
	done := make(chan error, 1)
	go func() {
		err := shutdownHTTP(ctx)
		close()
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
