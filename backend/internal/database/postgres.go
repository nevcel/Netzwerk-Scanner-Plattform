package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool erstellt einen gemeinsamen Connection-Pool
// für die Verbindung zwischen Backend und PostgreSQL.
func NewPostgresPool(ctx context.Context) (*pgxpool.Pool, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	database := getEnv("POSTGRES_DB", "network_scanner")
	user := getEnv("POSTGRES_USER", "network_scanner")
	password := os.Getenv("POSTGRES_PASSWORD")

	// Das Passwort wird bewusst nicht im Quellcode hinterlegt.
	if password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD ist nicht gesetzt")
	}

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		database,
	)

	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("datenbankverbindung konnte nicht erstellt werden: %w", err)
	}

	// Ping prüft direkt beim Start, ob PostgreSQL tatsächlich erreichbar ist.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgresql ist nicht erreichbar: %w", err)
	}

	return pool, nil
}

// getEnv liefert den Wert einer Umgebungsvariable.
// Ist kein Wert gesetzt, wird der angegebene Standard verwendet.
func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
