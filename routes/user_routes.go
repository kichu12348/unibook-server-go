package routes

import (
	"unibook-go/config"
	"unibook-go/handlers"
	"unibook-go/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(app *fiber.App, cfg *config.Config) {
	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	auth.Post("/register", handlers.RegisterUser(cfg))
	auth.Post("/verify-email", handlers.VerifyOtpAndLogin(cfg))
	auth.Post("/login", handlers.Login(cfg))
	auth.Post("/resend-otp", handlers.ResendOtp(cfg))
	auth.Post("/forgot-password", handlers.ForgotPassword(cfg))
	auth.Post("/verify-reset-otp", handlers.VerifyResetOtp)
	auth.Post("/reset-password", handlers.ResetPassword)
	auth.Get("/me", middleware.Protected(cfg), handlers.GetMe)
}
