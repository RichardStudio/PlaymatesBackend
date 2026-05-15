package repository

import (
	"playmates/components/playmates/models"
	"time"
)

func (r *Repository) InsertToken(userID int, username, hashedToken string, expiresAt time.Time, fingerprint string) error {
	_, err := r.db.Exec(`
		INSERT INTO refresh_tokens (user_id, username, token_hash, expires_at, fingerprint, revoked)
		VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, username, hashedToken, expiresAt, fingerprint, false)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetRefreshToken(refreshToken string) (models.RefreshToken, error) {
	var token models.RefreshToken

	err := r.db.QueryRow("SELECT id, user_id, username, token_hash, expires_at, revoked, fingerprint, created_at FROM refresh_tokens WHERE token_hash = $1", refreshToken).Scan(
		&token.ID, &token.UserID, &token.Username, &token.HashedToken, &token.ExpiresAt, &token.Revoked, &token.Fingerprint, &token.CreatedAt,
	)
	if err != nil {
		return models.RefreshToken{}, err
	}

	return token, nil
}

func (r *Repository) RevokeRefreshToken(refreshToken string) (bool, error) {
	res, err := r.db.Exec("UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1", refreshToken)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := res.RowsAffected()

	return rowsAffected > 0, nil
}
