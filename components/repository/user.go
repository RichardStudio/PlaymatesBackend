package repository

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"playmates/components/service/models"
)

func GetUser(id int, db *sql.DB) (models.User, error) {
	var user models.User
	var age sql.NullInt64
	var gender sql.NullString
	var aboutMe sql.NullString

	err := db.QueryRow("SELECT id, username, email, age, gender, games, about_me FROM users WHERE id = $1", id).Scan(
		&user.ID, &user.Username, &user.Email, &age, &gender, pq.Array(&user.Games), &aboutMe,
	)

	if err != nil {
		fmt.Println(err)
		return models.User{}, fmt.Errorf("cannot get user: %w", err)
	}

	if aboutMe.Valid {
		user.AboutMe = aboutMe.String
	}
	if gender.Valid {
		user.Gender = gender.String
	}
	if age.Valid {
		user.Age = int(age.Int64)
	}

	return user, nil
}

func SetUser(user models.User, db *sql.DB) error {
	_, err := db.Exec(
		`UPDATE users SET age = $1, gender = $2, games = $3, about_me = $4 WHERE id = $5`,
		user.Age, user.Gender, pq.Array(user.Games), user.AboutMe, user.ID,
	)

	if err != nil {
		fmt.Println(user)
		return err
	}

	return nil
}

func SearchUsers(minAge, maxAge, offset int, games []string, gender string, db *sql.DB) ([]models.User, int, error) {
	query := "SELECT id, username, email, age, gender, games, about_me FROM users WHERE 1=1"
	args := []interface{}{}

	if minAge > 0 && maxAge > 0 && minAge > maxAge {
		return nil, -1, fmt.Errorf("minimum age can't be greater than maximum age")
	}

	if minAge > 0 {
		query += fmt.Sprintf(" AND age >= $%d", len(args)+1)
		args = append(args, minAge)
	}

	if maxAge > 0 {
		query += fmt.Sprintf(" AND age <= $%d", len(args)+1)
		args = append(args, maxAge)
	}

	if gender != "" {
		query += fmt.Sprintf(" AND gender = $%d", len(args)+1)
		args = append(args, gender)
	}

	if len(games) > 0 {
		for _, game := range games {
			query += fmt.Sprintf(" AND games @> $%d", len(args)+1)
			args = append(args, pq.Array([]string{game}))
		}
	}

	total, err := CountSearch(minAge, maxAge, games, gender, db)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to count users: %w", err)
	}

	query += " ORDER BY id DESC"
	query += fmt.Sprintf(" LIMIT 20 OFFSET $%d", len(args)+1)
	args = append(args, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println(fmt.Sprintf("failed to execute query search: %v", err))
		return nil, -1, fmt.Errorf("failed to execute query for search users: %w", err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		var age sql.NullInt64
		var gen sql.NullString
		var aboutMe sql.NullString

		err = rows.Scan(
			&user.ID, &user.Username, &user.Email, &age, &gen, pq.Array(&user.Games), &aboutMe,
		)
		if err != nil {
			fmt.Println(fmt.Sprintf("failed to scan user row: %v", err))
			return nil, -1, fmt.Errorf("failed to scan user row: %w", err)
		}

		if aboutMe.Valid {
			user.AboutMe = aboutMe.String
		}
		if gen.Valid {
			user.Gender = gen.String
		}
		if age.Valid {
			user.Age = int(age.Int64)
		}

		users = append(users, user)
	}

	return users, total, nil
}

func CountSearch(minAge, maxAge int, games []string, gender string, db *sql.DB) (int, error) {
	query := "SELECT COUNT(*) FROM users WHERE 1=1"
	args := []interface{}{}

	if minAge > 0 && maxAge > 0 && minAge > maxAge {
		return -1, fmt.Errorf("minimum age can't be greater than maximum age")
	}

	if minAge > 0 {
		query += fmt.Sprintf(" AND age >= $%d", len(args)+1)
		args = append(args, minAge)
	}

	if maxAge > 0 {
		query += fmt.Sprintf(" AND age <= $%d", len(args)+1)
		args = append(args, maxAge)
	}

	if gender != "" {
		query += fmt.Sprintf(" AND gender = $%d", len(args)+1)
		args = append(args, gender)
	}

	if len(games) > 0 {
		for _, game := range games {
			query += fmt.Sprintf(" AND games @> $%d", len(args)+1)
			args = append(args, pq.Array([]string{game}))
		}
	}

	var total int
	err := db.QueryRow(query, args...).Scan(&total)
	if err != nil {
		return -1, fmt.Errorf("failed to execute query count: %w", err)
	}
	return total, nil
}
