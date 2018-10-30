package utils

import (
	"database/sql"
	"fmt"
)

func NewDb(settings DbSettings) (*sql.DB, error) {
	connectStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		settings.Host, settings.Port, settings.User, settings.Password, settings.Database)

	db, err := sql.Open("postgres", connectStr)
	if err != nil {
		return nil, err
	}

	return db, db.Ping()
}
