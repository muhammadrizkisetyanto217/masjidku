package controller

import (
	"log"

	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"masjidku/internals/configs"
	modelAuth "masjidku/internals/features/users/auth/models"
	modelUser "masjidku/internals/features/users/user/models"

	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
}

// ============================ REGISTER ============================
func (ac *AuthController) Register(c *fiber.Ctx) error {
	var input modelUser.UserModel
	if err := c.BodyParser(&input); err != nil {
		log.Printf("[ERROR] Failed to parse request body: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}
	if err := input.Validate(); err != nil {
		log.Printf("[ERROR] Validation failed: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] Failed to hash password: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to secure password"})
	}
	input.Password = string(passwordHash)
	if err := ac.DB.Create(&input).Error; err != nil {
		log.Printf("[ERROR] Failed to save user to database: %v", err)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return c.Status(400).JSON(fiber.Map{"error": "Email already registered"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to register user"})
	}
	log.Printf("[SUCCESS] User registered: ID=%v, Email=%s", input.ID, input.Email)
	return c.Status(201).JSON(fiber.Map{"message": "User registered successfully"})
}

// ============================ LOGIN ============================
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var input struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	var user modelUser.UserModel
	if err := ac.DB.Where("email = ? OR user_name = ?", input.Identifier, input.Identifier).First(&user).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid email, username, or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid email, username, or password"})
	}

	// Generate Access Token (15 menit)
	accessExp := time.Now().Add(15 * time.Minute)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":        user.ID.String(),
		"user_name": user.UserName,
		"role":      user.Role,
		"exp":       accessExp.Unix(),
	})
	accessTokenString, err := accessToken.SignedString([]byte(configs.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate access token"})
	}

	// Generate Refresh Token (7 hari)
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.ID.String(),
		"exp": refreshExp.Unix(),
	})
	refreshTokenString, err := refreshToken.SignedString([]byte(configs.JWTRefreshSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate refresh token"})
	}

	// Simpan refresh token ke DB
	rt := modelAuth.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: refreshExp,
	}
	if err := ac.DB.Create(&rt).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store refresh token"})
	}

	user.Password = ""
	// Set refresh_token ke dalam HttpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict", // bisa diubah ke Lax jika butuh cross-domain
		Expires:  refreshExp,
	})

	// Kirim access_token dan user data saja
	return c.JSON(fiber.Map{
		"access_token": accessTokenString,
		"user": fiber.Map{
			"id":        user.ID,
			"user_name": user.UserName,
			"email":     user.Email,
			"role":      user.Role,
		},
	})

}

func (ac *AuthController) RefreshToken(c *fiber.Ctx) error {
	// 1. Ambil refresh_token dari cookie
	oldToken := c.Cookies("refresh_token")
	if oldToken == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Refresh token not found"})
	}

	// 2. Verifikasi JWT refresh token
	token, err := jwt.Parse(oldToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(configs.JWTRefreshSecret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid or expired refresh token"})
	}

	// 3. Ambil claims (user ID)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["id"] == nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token claims"})
	}
	userID := claims["id"].(string)

	// 4. Cek token di database (validasi)
	var stored modelAuth.RefreshToken
	if err := ac.DB.Where("token = ?", oldToken).First(&stored).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Refresh token not registered"})
	}
	if time.Now().After(stored.ExpiresAt) {
		ac.DB.Delete(&stored)
		return c.Status(401).JSON(fiber.Map{"error": "Refresh token expired"})
	}

	// 5. Ambil user
	var user modelUser.UserModel
	if err := ac.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// 6. Hapus refresh token lama
	ac.DB.Delete(&stored)

	// 7. Generate access token baru
	accessExp := time.Now().Add(15 * time.Minute)
	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":        user.ID.String(),
		"user_name": user.UserName,
		"role":      user.Role,
		"exp":       accessExp.Unix(),
	})
	accessTokenString, err := newAccessToken.SignedString([]byte(configs.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate access token"})
	}

	// 8. Generate refresh token baru
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.ID.String(),
		"exp": refreshExp.Unix(),
	})
	newRefreshTokenString, err := newRefreshToken.SignedString([]byte(configs.JWTRefreshSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate new refresh token"})
	}

	// 9. Simpan refresh token baru ke DB
	newRT := modelAuth.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshTokenString,
		ExpiresAt: refreshExp,
	}
	if err := ac.DB.Create(&newRT).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store new refresh token"})
	}

	// 10. Set cookie refresh token baru
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshTokenString,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  refreshExp,
	})

	// 11. Return access token baru
	return c.JSON(fiber.Map{
		"access_token": accessTokenString,
	})
}
