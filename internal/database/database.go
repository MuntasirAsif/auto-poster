package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const defaultDatabaseURL = "postgres://autoposter:autoposter@localhost:5432/auto_poster"

//go:embed migrations/*.sql
var embedMigrations embed.FS

func databaseURL() string {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	return defaultDatabaseURL
}

func Connect() (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL())
	if err != nil {
		return nil, err
	}

	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	log.Println("Database connected successfully")

	return pool, nil
}

func ApplyMigrations() error {
	connConfig, err := pgx.ParseConfig(databaseURL())
	if err != nil {
		return err
	}
	connConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec

	db, err := sql.Open("pgx", stdlib.RegisterConnConfig(connConfig))
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return err
	}

	results, err := provider.Up(context.Background())
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	for _, res := range results {
		log.Printf("migration applied: %s", res.Source.Path)
	}

	return nil
}