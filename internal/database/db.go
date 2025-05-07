package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func ConnectDB(dataSourse string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourse)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	// testing the connection

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err) // ensures db is reachable 
	}
	
	return db, nil
}