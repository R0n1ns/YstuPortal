package main

import (
	"YstuPortal/internal/config"
	"YstuPortal/internal/logic"
	"YstuPortal/internal/repository/cache/redis"
	"YstuPortal/internal/repository/userProvider"
	"YstuPortal/internal/repository/userStorage/db"
	"YstuPortal/internal/server"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	storage := db.NewUserStorage(cfg.DatabaseURL)
	defer storage.Close()
	parser := userProvider.NewUserParser()
	defer parser.Close()

	var cache logic.EstimationsCache
	if cfg.RedisURL != "" {
		redisCache, err := redis.NewEstimationsCache(cfg.RedisURL)
		if err != nil {
			log.Printf("redis cache disabled: %v", err)
		} else {
			cache = redisCache
			defer func() { _ = redisCache.Close() }()
		}
	}

	var dataManager *logic.UserManager
	if cache != nil {
		dataManager, err = logic.NewUserManagerWithCache(parser, storage, cache, cfg.CacheTTL)
	} else {
		dataManager, err = logic.NewUserManager(parser, storage)
	}
	if err != nil {
		log.Fatalf("init user manager: %v", err)
	}

	app := server.New(cfg, *dataManager)

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
