package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing Authorization header"})
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid Authorization header"})
	}

	token, err := h.service.ParseToken(tokenString)
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	return c.Next()
}
