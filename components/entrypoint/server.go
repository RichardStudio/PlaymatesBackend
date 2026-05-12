package entrypoint

import (
	"playmates/components/playmates/handler"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func New(handler *handler.Handler) *fiber.App {
	app := fiber.New()

	// Добавляем CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",                           // Разрешаем запросы только с этого домена
		AllowMethods:     "GET, POST, PUT, DELETE",      // Разрешенные методы
		AllowHeaders:     "Content-Type, Authorization", // Разрешенные заголовки
		AllowCredentials: false,                         // Разрешаем передачу куки и авторизационных заголовков
	}))

	// Routes
	app.Post("/register", handler.Register)

	app.Post("/login", handler.Login)

	app.Get("/protected", handler.AuthMiddleware, handler.Protected)

	app.Get("/profile", handler.AuthMiddleware, handler.GetProfile)
	app.Put("/profile", handler.AuthMiddleware, handler.UpdateProfile)

	app.Get("/search", handler.AuthMiddleware, handler.Search)

	app.Get("/profile/:id", handler.AuthMiddleware, handler.GetProfileById)

	app.Get("/connection-manager/:id", handler.AuthMiddleware, handler.GetChatMessages)

	app.Get("/ws/", websocket.New(handler.WebSocketConnect))

	app.Get("/messages", handler.AuthMiddleware, handler.GetMessages)

	return app
}
