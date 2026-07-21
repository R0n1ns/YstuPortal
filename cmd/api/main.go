package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/R0n1ns/YstuPortal/internal/config"
	"github.com/R0n1ns/YstuPortal/internal/domain"
	"github.com/R0n1ns/YstuPortal/internal/logic"
	"github.com/R0n1ns/YstuPortal/internal/repository/cache/redis"
	"github.com/R0n1ns/YstuPortal/internal/repository/userProvider"
	"github.com/R0n1ns/YstuPortal/internal/repository/userStorage/db"
	"github.com/R0n1ns/YstuPortal/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Printf("application stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	storage, err := db.NewUserStorage(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer storage.Close()

	var provider domain.UserProvider
	if cfg.DemoMode {
		log.Print("demo provider enabled")
		provider = userProvider.NewDemoProvider()
	} else {
		parser := userProvider.NewUserParser(
			cfg.UpstreamBaseURL,
			cfg.UpstreamCode,
			cfg.UpstreamTimeout,
			cfg.SessionTTL,
		)
		defer parser.Close()
		provider = parser
	}

	var manager *logic.UserManager
	if cfg.RedisURL != "" {
		cache, cacheErr := redis.NewGradesCache(cfg.RedisURL)
		if cacheErr != nil {
			log.Printf("redis cache disabled: %v", cacheErr)
		} else {
			defer func() { _ = cache.Close() }()
			manager, err = logic.NewUserManagerWithCache(provider, storage, cache, cfg.CacheTTL)
		}
	}
	if manager == nil && err == nil {
		manager, err = logic.NewUserManager(provider, storage)
	}
	if err != nil {
		return err
	}

	app := server.New(cfg, manager)
	errCh := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on :%s", cfg.Port)
		errCh <- app.Listen(":" + cfg.Port)
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-shutdownCtx.Done():
		log.Print("shutting down HTTP server")
		if err := app.Shutdown(); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
}
