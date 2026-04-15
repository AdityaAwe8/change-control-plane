package main

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
)

func main() {
	cfg := common.LoadConfig()
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := storage.ApplyMigrations(context.Background(), db, filepath.Join("db", "migrations")); err != nil {
		log.Fatal(err)
	}

	log.Printf("migrations applied successfully")
}
