package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"unibook-go/config"
	"unibook-go/database"
	db "unibook-go/database/db"
	"unibook-go/middleware"
	"unibook-go/util"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type RegisterPayload struct {
	FullName  string    `json:"fullName"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Role      string    `json:"role"`
	CollegeID uuid.UUID `json:"collegeId"`
	ForumID   uuid.UUID `json:"forumId,omitempty"`
}

type VerifyOtpPayload struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResendOtpOrPasswordResetPayload struct {
	Email string `json:"email"`
}

type VerifyForgotPasswordOtpPayload struct {
	Email string `json:"email"`
	Otp   string `json:"otp"`
}

type ResetPasswordPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Otp      string `json:"otp"`
}

type College struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func RegisterUser(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		payload := new(RegisterPayload)
		if err := c.BodyParser(payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
		}

		// /"student", "teacher", "forum_head"
		if payload.Role != "student" && payload.Role != "teacher" && payload.Role != "forum_head" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role provided"})
		}

		if payload.Email == "" || payload.Password == "" || payload.CollegeID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Credentials"})
		}

		queries := db.New(database.DB)

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(payload.Password), 10)
		approvalStatus := db.ApprovalStatusPending
		if db.UserRole(payload.Role) == db.UserRoleStudent {
			approvalStatus = db.ApprovalStatusApproved
		}

		userParams := db.CreateUserParams{
			FullName:       payload.FullName,
			Email:          payload.Email,
			PasswordHash:   string(hashedPassword),
			Role:           db.UserRole(payload.Role),
			CollegeID:      payload.CollegeID,
			ApprovalStatus: approvalStatus,
		}
		newUser, err := queries.CreateUser(c.Context(), userParams)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user account."})
		}

		if db.UserRole(payload.Role) == db.UserRoleForumHead && payload.ForumID != uuid.Nil {
			forumHeadParams := db.CreateForumHeadParams{
				UserID:  newUser.ID,
				ForumID: payload.ForumID,
			}
			_, _ = queries.CreateForumHead(c.Context(), forumHeadParams)
		}

		otp := fmt.Sprintf("%04d", rand.Intn(10000))
		log.Printf("Generated OTP for %s: %s", newUser.Email, otp)
		hashedOtp, _ := bcrypt.GenerateFromPassword([]byte(otp), 10)

		verificationParams := db.SetUserEmailVerificationDetailsParams{
			ID:                       newUser.ID,
			EmailVerificationToken:   pgtype.Text{String: string(hashedOtp), Valid: true},
			EmailVerificationExpires: pgtype.Timestamp{Time: time.Now().Add(10 * time.Minute), Valid: true},
		}
		_ = queries.SetUserEmailVerificationDetails(c.Context(), verificationParams)

		go util.SendOtpEmail(cfg, newUser.Email, otp)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Registration successful. Please check your email for a verification code.",
		})
	}
}

func VerifyOtpAndLogin(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		payload := new(VerifyOtpPayload)
		if err := c.BodyParser(payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
		}

		queries := db.New(database.DB)
		user, err := queries.GetUserByEmail(c.Context(), payload.Email)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid OTP or request has expired."})
		}

		if user.IsEmailVerified {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email is already verified."})
		}
		if !user.EmailVerificationToken.Valid || time.Now().After(user.EmailVerificationExpires.Time) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid OTP or request has expired."})
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.EmailVerificationToken.String), []byte(payload.OTP))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid OTP."})
		}

		updatedUser, err := queries.VerifyUserEmail(c.Context(), user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to verify email."})
		}

		if updatedUser.ApprovalStatus != db.ApprovalStatusApproved {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Your account has been verified, but is pending approval by the college admin.",
				"code":    "PENDING_APPROVAL",
			})
		}

		claims := jwt.MapClaims{
			"id":        updatedUser.ID.String(),
			"role":      updatedUser.Role,
			"collegeId": updatedUser.CollegeID.String(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		t, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to generate token")
		}

		return c.JSON(fiber.Map{"message": "Email verified successfully.", "token": t})
	}
}

func Login(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body LoginPayload
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
		}
		if body.Password == "" || body.Email == "" {
			return c.Status(400).JSON(fiber.Map{"error": "invalid credentials"})
		}

		queries := db.New(database.DB)

		superAdmin, err := queries.GetSuperAdminByEmail(c.Context(), body.Email)

		if err == nil {
			err := bcrypt.CompareHashAndPassword([]byte(superAdmin.PasswordHash), []byte(body.Password))
			if err == nil {
				claims := jwt.MapClaims{
					"id":   superAdmin.ID,
					"role": "super_admin",
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				t, _ := token.SignedString([]byte(cfg.JWTSecret))
				return c.JSON(fiber.Map{"token": t})
			}
		}

		user, err := queries.GetUserByEmail(c.Context(), body.Email)

		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "invalid email"})
		}

		if !user.IsEmailVerified {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Your account is not verified. Please complete the OTP verification process.",
				"code":  "NOT_VERIFIED",
				"email": user.Email,
			})
		}

		if user.ApprovalStatus == db.ApprovalStatusRejected {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Your account has been rejected by the college admin.",
				"code":  "ACCOUNT_REJECTED",
			})
		}
		if user.ApprovalStatus != db.ApprovalStatusApproved {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Your account is pending approval from the college admin.",
				"code":  "PENDING_APPROVAL",
			})
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials."})
		}

		claims := jwt.MapClaims{
			"id":        user.ID.String(),
			"role":      user.Role,
			"collegeId": user.CollegeID.String(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		t, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to generate token")
		}

		return c.JSON(fiber.Map{"token": t})
	}
}

func ResendOtp(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload ResendOtpOrPasswordResetPayload

		if err := c.BodyParser(&payload); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Email is required"})
		}

		queries := db.New(database.DB)

		user, err := queries.GetUserByEmail(c.Context(), payload.Email)

		if err != nil {
			return c.JSON(fiber.Map{"message": "otp send"})
		}

		otp := fmt.Sprintf("%04d", rand.Intn(10000))
		hashedOtp, _ := bcrypt.GenerateFromPassword([]byte(otp), 10)
		verificationParam := db.SetUserEmailVerificationDetailsParams{
			ID:                       user.ID,
			EmailVerificationToken:   pgtype.Text{String: string(hashedOtp), Valid: true},
			EmailVerificationExpires: pgtype.Timestamp{Time: time.Now().Add(10 * time.Minute), Valid: true},
		}

		err = queries.SetUserEmailVerificationDetails(c.Context(), verificationParam)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error"})
		}

		go util.SendOtpEmail(cfg, user.Email, otp)

		return c.JSON(fiber.Map{
			"message": "A new verification code has been sent to your email.",
		})

	}
}

func ForgotPassword(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload ResendOtpOrPasswordResetPayload

		if err := c.BodyParser(&payload); err != nil {
			return c.Status(403).JSON(fiber.Map{"error": "Invalid Json"})
		}

		if payload.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email is Required"})
		}

		queries := db.New(database.DB)

		user, err := queries.GetUserByEmail(c.Context(), payload.Email)

		if err != nil {
			return c.JSON(fiber.Map{"message": "otp send"})
		}

		otp := fmt.Sprintf("%04d", rand.Intn(10000))

		hashedOtp, _ := bcrypt.GenerateFromPassword([]byte(otp), 10)

		forgotPasswordParams := db.SetUserPasswordResetDetailsParams{
			ID:                   user.ID,
			PasswordResetToken:   pgtype.Text{String: string(hashedOtp), Valid: true},
			PasswordResetExpires: pgtype.Timestamp{Time: time.Now().Add(10 * time.Minute), Valid: true},
		}

		err = queries.SetUserPasswordResetDetails(c.Context(), forgotPasswordParams)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error"})
		}

		go util.SendOtpEmail(cfg, user.Email, otp)

		return c.JSON(fiber.Map{
			"message": "A password reset code has been sent to your email.",
		})
	}
}

func VerifyResetOtp(c *fiber.Ctx) error {
	var payload VerifyForgotPasswordOtpPayload

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(403).JSON(fiber.Map{"error": "Invalid Json"})
	}

	if payload.Email == "" || payload.Otp == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "payload is Required"})
	}

	queries := db.New(database.DB)

	user, err := queries.GetUserByEmail(c.Context(), payload.Email)

	if err != nil || !user.PasswordResetToken.Valid || user.PasswordResetToken.String == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid or expired reset token"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordResetToken.String), []byte(payload.Otp))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid OTP."})
	}

	return c.JSON(fiber.Map{
		"message": "OTP verified successfully.",
	})
}

func ResetPassword(c *fiber.Ctx) error {
	var payload ResetPasswordPayload

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(403).JSON(fiber.Map{"error": "Invalid Json"})
	}

	if payload.Email == "" || payload.Otp == "" || payload.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "payload is Required"})
	}

	queries := db.New(database.DB)

	user, err := queries.GetUserByEmail(c.Context(), payload.Email)

	if err != nil || !user.PasswordResetToken.Valid || user.PasswordResetToken.String == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid or expired reset token"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordResetToken.String), []byte(payload.Otp))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid OTP."})
	}

	newPassword, _ := bcrypt.GenerateFromPassword([]byte(payload.Password), 10)
	updatePasswordParam := db.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: string(newPassword),
	}

	err = queries.UpdateUserPassword(c.Context(), updatePasswordParam)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully.",
	})
}

func GetMe(c *fiber.Ctx) error {
	authUser := c.Locals("authUser").(middleware.AuthUser)

	queries := db.New(database.DB)

	if authUser.Role == "super_admin" {
		adminProfile, err := queries.GetSuperAdminByID(c.Context(), authUser.ID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Super admin profile not found."})
		}
		return c.JSON(fiber.Map{
			"id":        adminProfile.ID,
			"fullName":  adminProfile.FullName,
			"email":     adminProfile.Email,
			"createdAt": adminProfile.CreatedAt,
			"role":      "super_admin",
		})
	}

	userProfile, err := queries.GetUserByID(c.Context(), authUser.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found."})
	}

	var collegeObj College

	json.Unmarshal(userProfile.College, &collegeObj)

	return c.JSON(fiber.Map{
		"id":              userProfile.ID,
		"fullName":        userProfile.FullName,
		"email":           userProfile.Email,
		"role":            userProfile.Role,
		"collegeId":       userProfile.CollegeId,
		"approvalStatus":  userProfile.ApprovalStatus,
		"isEmailVerified": userProfile.IsEmailVerified,
		"createdAt":       userProfile.CreatedAt,
		"college":         collegeObj,
		"forumHeads":      userProfile.ForumHeads,
	})
}
