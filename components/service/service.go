package service

import "database/sql"

type Service struct {
	DB        *sql.DB
	JwtSecret string
}

func New(db *sql.DB, jwtSecret string) *Service {
	return &Service{
		DB:        db,
		JwtSecret: jwtSecret,
	}
}
