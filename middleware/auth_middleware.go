package middleware

import (
	"unibook-go/config"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthUser struct {
	ID        uuid.UUID
	Role      string
	CollegeID *uuid.UUID
}

func Protected(cfg *config.Config) fiber.Handler {
	// Create a new JWT middleware handler
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(cfg.JWTSecret)},

		// This function is called after the token is successfully validated.
		SuccessHandler: func(c *fiber.Ctx) error {
			// Get the decoded token from the context
			token := c.Locals("user").(*jwt.Token)
			claims := token.Claims.(jwt.MapClaims)

			// Parse the ID
			id, err := uuid.Parse(claims["id"].(string))
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
			}

			// Create our AuthUser struct
			authUser := AuthUser{
				ID:   id,
				Role: claims["role"].(string),
			}

			// Check for the optional collegeId
			if collegeIdClaim, ok := claims["collegeId"]; ok {
				collegeId, err := uuid.Parse(collegeIdClaim.(string))
				if err == nil {
					authUser.CollegeID = &collegeId
				}
			}

			// Store the structured user data in the context for the next handler
			c.Locals("authUser", authUser)
			return c.Next()
		},

		// This function is called if token validation fails
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Invalid or missing token",
			})
		},
	})
}
