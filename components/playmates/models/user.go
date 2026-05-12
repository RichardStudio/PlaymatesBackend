package models

type User struct {
	ID           int      `json:"id"`
	Age          int      `json:"age"`
	Gender       string   `json:"gender"`
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"password_hash"`
	AboutMe      string   `json:"about_me"`
	Games        []string `json:"games"`
}
