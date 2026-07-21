package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	path := flag.String("path", "migrations", "path to migrations")
	databaseURL := flag.String("database", os.Getenv("DATABASE_URL"), "database url")
	direction := flag.String("direction", "up", "up or down")
	steps := flag.Int("steps", 0, "number of steps (0 = all)")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	m, err := migrate.New("file://"+*path, *databaseURL)
	if err != nil {
		log.Fatalf("create migrate: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	switch *direction {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	case "version":
		version, dirty, vErr := m.Version()
		if vErr != nil {
			log.Fatalf("version: %v", vErr)
		}
		fmt.Printf("version=%d dirty=%v\n", version, dirty)
		return
	default:
		log.Fatalf("unknown direction: %s", *direction)
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate %s: %v", *direction, err)
	}
}
