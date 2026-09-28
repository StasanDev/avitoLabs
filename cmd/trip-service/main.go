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
	defer pool.Close()
	log.Printf("connected to db")

	tripRepository := repo.NewTripRepository(pool, cfg.DB.QueryTimeout)
	tripStatusHistoryRepository := repo.NewTripStatusRepository(pool, cfg.DB.QueryTimeout)
	txManager := repo.NewTxManager(pool)
	tripService := usecase.NewTripService(tripRepository, tripStatusHistoryRepository, txManager)
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
		shutdownCtx, cancelShutdown := context.WithTimeout(ctx, cfg.ShutdownTimeout)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
	log.Printf("service stopped")
}
