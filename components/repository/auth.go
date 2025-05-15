package repository

import (
	"database/sql"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"playmates/components/service/models"
	"strings"
	"time"
)

func Register(username, email, password string, db *sql.DB) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = db.Exec(
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

func Login(email, password, jwtSecret string, db *sql.DB) (string, error) {
	var user models.User

	email = strings.ToLower(email)
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE email = $1", email).Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		fmt.Println(fmt.Sprintf("err login email: %s, err: %w", email, err))
		return "", fmt.Errorf("invalid email or password")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"user_id":  user.ID,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	jwtString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt: %w", err)
	}

	return jwtString, nil
}

func GetIdFromToken(tokenString, jwtSecret string) (int, error) {
	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return -1, fmt.Errorf("failed to parse token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return -1, fmt.Errorf("failed to parse claims")
	}

	UserIdFloat, ok := claims["user_id"].(float64)
	if !ok {
		return -1, fmt.Errorf("no user id found")
	}

	UserId := int(UserIdFloat)

	return UserId, nil
}
