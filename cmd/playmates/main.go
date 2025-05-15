package main

import (
	"fmt"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"playmates/components/chat"
	"playmates/components/db"
	"playmates/components/repository"
	"playmates/components/service"
	"playmates/components/service/config"
	"strconv"
	"strings"
)

func main() {
	configPath := "config/config.yaml"

	cfg, err := config.New(configPath)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error loading config: %w", err))
		return
	}

	db, err := db.ConnectPostgres(cfg.DbConnStr)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error connecting to database: %w", err))
		return
	}

	service := service.New(db, cfg.JwtSecret)

	app := fiber.New()

	connenctionManager := chat.NewConnectionManager()

	// Добавляем CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",                           // Разрешаем запросы только с этого домена
		AllowMethods:     "GET, POST, PUT, DELETE",      // Разрешенные методы
		AllowHeaders:     "Content-Type, Authorization", // Разрешенные заголовки
		AllowCredentials: false,                         // Разрешаем передачу куки и авторизационных заголовков
	}))

	// Routes
	app.Post("/register", func(c *fiber.Ctx) error {
		type Request struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var req Request
		if err = c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := repository.Register(req.Username, req.Email, req.Password, db); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error creating a new user. Perhaps such an Email or Username is already in use."})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user registered"})
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		type Request struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		token, err := repository.Login(req.Email, req.Password, cfg.JwtSecret, db)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		id, err := repository.GetIdFromToken(token, cfg.JwtSecret)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		user, err := repository.GetUser(id, db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"token": token,
			"user":  user,
		})
	})

	app.Get("/protected", service.AuthMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "protected page"})
	})

	app.Get("/profile", service.AuthMiddleware, func(c *fiber.Ctx) error {
		token := c.Get("Authorization")

		userId, err := repository.GetIdFromToken(token, cfg.JwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := repository.GetUser(userId, db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(user)
	})

	app.Put("/profile", service.AuthMiddleware, func(c *fiber.Ctx) error {
		token := c.Get("Authorization")

		userId, err := repository.GetIdFromToken(token, cfg.JwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := repository.GetUser(userId, db)
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
			fmt.Println(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		user.Age = updateData.Age
		user.Gender = updateData.Gender
		user.Games = updateData.Games
		user.AboutMe = updateData.AboutMe

		err = repository.SetUser(user, db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "user updated"})
	})
	app.Get("/search", service.AuthMiddleware, func(c *fiber.Ctx) error {
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
				fmt.Println(fmt.Sprintf("invalid min age: %v", err))
			}
		}
		if maxAgeStr != "" {
			maxAge, err = strconv.Atoi(maxAgeStr)
			if err != nil {
				fmt.Println(fmt.Sprintf("invalid max age: %v", err))
			}
		}
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				fmt.Println(fmt.Sprintf("invalid offset: %v", err))
			}
		}
		var games []string
		if gamesStr != "" {
			games = strings.Split(gamesStr, ",")
		}

		users, total, err := repository.SearchUsers(minAge, maxAge, offset, games, gender, db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"total": total,
			"users": users,
		})
	})

	app.Get("/profile/:id", service.AuthMiddleware, func(c *fiber.Ctx) error {
		// Получаем ID пользователя из параметров запроса
		userID, err := strconv.Atoi(c.Params("id"))
		if err != nil || userID <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		// Получаем данные пользователя из базы данных
		user, err := repository.GetUser(userID, db)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}

		return c.JSON(user)
	})

	app.Get("/chat/:id", service.AuthMiddleware, func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		currentUserID, err := repository.GetIdFromToken(token, cfg.JwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		otherUserID, err := strconv.Atoi(c.Params("id"))
		if err != nil || otherUserID <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		// Получаем все сообщения между двумя пользователями
		messages, err := repository.GetMessages(currentUserID, otherUserID, db)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := repository.GetUser(otherUserID, db)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}

		return c.JSON(fiber.Map{
			"messages": messages,
			"user":     user,
		})
	})

	app.Get("/ws/", websocket.New(func(c *websocket.Conn) {
		fmt.Println("пришло")
		token := c.Query("token")
		userID, err := repository.GetIdFromToken(token, cfg.JwtSecret)
		if err != nil {
			fmt.Println("error getting user ID: ", err)
			c.WriteMessage(websocket.CloseMessage, []byte(fmt.Sprintf("err getting userID: %v", err)))
			return
		}
		chat.HandleWebSocket(c, db, connenctionManager, userID)
	}))

	app.Get("/messages", service.AuthMiddleware, func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		userID, err := repository.GetIdFromToken(token, cfg.JwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		chats, err := repository.GetUserChats(userID, db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"chats": chats})
	})

	fmt.Println(app.Listen(":8080"))
}
