package repository

import (
	"fmt"
	"playmates/components/playmates/models"

	"github.com/lib/pq"
)

func (r *Repository) Register(username, email, hashedPassword string) error {
	_, err := r.db.Exec(
		"INSERT INTO users (username, email, password_hash, age, gender, about_me, games) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		username,
		email,
		string(hashedPassword),
		0,
		"",
		"",
		pq.Array([]string{}),
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *Repository) Login(email string) (*models.User, error) {
	var user models.User

	err := r.db.QueryRow("SELECT id, password_hash FROM users WHERE email = $1", email).Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("sql error: %w", err)
	}

	return &user, nil
}
