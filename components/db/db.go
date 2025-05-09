package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func ConnectPostgres(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging db: %w", err)
	}

	return db, nil
}
