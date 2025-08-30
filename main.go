package main

import (
	"context"
	"log"

	"unibook-go/config"
	"unibook-go/database"
	"unibook-go/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	database.Connect(cfg.DatabaseURL)

	app := fiber.New()

	// /api/v1/auth
	routes.SetupAuthRoutes(app, cfg)

	app.Get("/", func(c *fiber.Ctx) error {
		if err := database.DB.Ping(context.Background()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Database connection failed",
			})
		}

		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Database connected successfully",
		})
	})

	log.Printf("Server is running on http://%s", cfg.ServerAddr)
	if err := app.Listen(cfg.ServerAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
