package handler

import (
	"database/sql"
	"fmt"
	"log"
	"playmates/components/playmates/config"
	"playmates/components/playmates/service"
	"strconv"
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	cfg     *config.Config
	db      *sql.DB
	service *service.Service
}

func New(cfg *config.Config, db *sql.DB, service *service.Service) *Handler {
	return &Handler{
		cfg:     cfg,
		db:      db,
		service: service,
	}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	type Request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.Register(req.Username, req.Email, req.Password); err != nil {
		log.Println("error registering user: ", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error creating a new user. Perhaps such an Email or Username is already in use."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user registered"})
}

func (h *Handler) Login(c *fiber.Ctx) error {
	type Request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	fingerprint := getFingerprint(c)
	token, refresh, refreshExpires, err := h.service.Login(req.Email, req.Password, fingerprint)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id, err := h.service.GetIdFromToken(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	user, err := h.service.GetUser(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	c.Cookie(&fiber.Cookie{
		Name:        "refresh_token",
		Value:       refresh,
		Path:        "/refresh",
		Expires:     refreshExpires,
		Secure:      false,
		HTTPOnly:    true,
		SameSite:    "Strict",
		SessionOnly: false,
	})

	return c.JSON(fiber.Map{
		"token": token,
		"user":  user,
	})
}

func (h *Handler) Protected(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "protected page"})
}

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	token := c.Get("Authorization")

	userId, err := h.service.GetIdFromToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.service.GetUser(userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(user)
}

func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	token := c.Get("Authorization")

	userId, err := h.service.GetIdFromToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.service.GetUser(userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	type ProfileUpdate struct {
		Age     int      `json:"age"`
		Gender  string   `json:"gender"`
		Games   []string `json:"games"`
		AboutMe string   `json:"about_me"`
	}

	var updateData ProfileUpdate
	if err := c.BodyParser(&updateData); err != nil {
		log.Printf("err updating profile, err: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	user.Age = updateData.Age
	user.Gender = updateData.Gender
	user.Games = updateData.Games
	user.AboutMe = updateData.AboutMe

	err = h.service.SetUser(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user updated"})
}

func (h *Handler) GetProfileById(c *fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil || userID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Получаем данные пользователя из базы данных
	user, err := h.service.GetUser(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func (h *Handler) Search(c *fiber.Ctx) error {
	var err error
	minAgeStr := c.Query("minAge")
	maxAgeStr := c.Query("maxAge")
	gamesStr := c.Query("games")
	gender := c.Query("gender")
	offsetStr := c.Query("offset")

	minAge := -1
	maxAge := -1
	offset := 0

	if minAgeStr != "" {
		minAge, err = strconv.Atoi(minAgeStr)
		if err != nil {
			log.Println(fmt.Sprintf("invalid min age: %v", err))
		}
	}
	if maxAgeStr != "" {
		maxAge, err = strconv.Atoi(maxAgeStr)
		if err != nil {
			log.Println(fmt.Sprintf("invalid max age: %v", err))
		}
	}
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			log.Println(fmt.Sprintf("invalid offset: %v", err))
		}
	}
	var games []string
	if gamesStr != "" {
		games = strings.Split(gamesStr, ",")
	}

	users, total, err := h.service.SearchUsers(minAge, maxAge, offset, games, gender)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"total": total,
		"users": users,
	})
}

func (h *Handler) GetChatMessages(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	currentUserID, err := h.service.GetIdFromToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	otherUserID, err := strconv.Atoi(c.Params("id"))
	if err != nil || otherUserID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Получаем все сообщения между двумя пользователями
	messages, err := h.service.GetMessages(currentUserID, otherUserID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.service.GetUser(otherUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"user":     user,
	})
}

func (h *Handler) WebSocketConnect(c *websocket.Conn) {
	token := c.Query("token")
	userID, err := h.service.GetIdFromToken(token)
	if err != nil {
		fmt.Println("error getting user ID: ", err)
		c.WriteMessage(websocket.CloseMessage, []byte(fmt.Sprintf("err getting userID: %v", err)))
		return
	}
	h.service.HandleWebSocket(c, userID)
}

func (h *Handler) GetMessages(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	userID, err := h.service.GetIdFromToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	chats, err := h.service.GetUserChats(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"chats": chats})
}

func getFingerprint(c *fiber.Ctx) string {
	ip := c.IP()
	agent := c.Get("User-Agent")

	return fmt.Sprintf("%s|%s", ip, agent)
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	refToken := c.Cookies("refresh_token")
	if refToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no refresh token"})
	}

	fingerprint := getFingerprint(c)

	ok, accessToken, newRefToken, expiresAt, err := h.service.Refresh(refToken, fingerprint)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{})
	}

	c.Cookie(&fiber.Cookie{
		Name:        "refresh_token",
		Value:       newRefToken,
		Path:        "/refresh",
		Expires:     expiresAt,
		Secure:      false,
		HTTPOnly:    true,
		SameSite:    "Strict",
		SessionOnly: false,
	})

	return c.JSON(fiber.Map{
		"token": accessToken,
	})
}
