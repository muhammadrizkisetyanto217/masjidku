// 🔥 LOGOUT USER
package controller

import (
	"log"
	"os"
	"strings"
	"time"

	modelAuth "masjidku/internals/features/users/auth/models"

	"github.com/gofiber/fiber/v2"
)

// 🔥 LOGOUT USER
func (ac *AuthController) Logout(c *fiber.Ctx) error {
	// ✅ 0. Pastikan JWT_SECRET ada di env
	if os.Getenv("JWT_SECRET") == "" {
		log.Println("[ERROR] JWT_SECRET tidak ditemukan di environment")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal Server Error - Missing JWT Secret",
		})
	}

	// ✅ 1. Ambil access token dari Authorization header
	authHeader := c.Get("Authorization")
	log.Println("[DEBUG] Authorization Header:", authHeader)

	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized - No token provided in header",
		})
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized - Invalid token format (expected 'Bearer <token>')",
		})
	}

	tokenString := tokenParts[1]

	// ✅ 2. Cek apakah token sudah diblacklist
	var existingToken modelAuth.TokenBlacklist
	if err := ac.DB.Where("token = ?", tokenString).First(&existingToken).Error; err == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Access token already blacklisted",
		})
	}

	// ✅ 3. Simpan token ke blacklist (anggap expired-nya sama dengan access token default, misal 96 jam)
	blacklistToken := modelAuth.TokenBlacklist{
		Token:     tokenString,
		ExpiredAt: time.Now().Add(96 * time.Hour),
	}
	if err := ac.DB.Create(&blacklistToken).Error; err != nil {
		log.Printf("[ERROR] Failed to blacklist token: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal Server Error - Failed to blacklist token",
		})
	}

	// ✅ 4. Hapus refresh_token dari DB jika ada
	refreshToken := c.Cookies("refresh_token")
	if refreshToken != "" {
		if err := ac.DB.Where("token = ?", refreshToken).Delete(&modelAuth.RefreshToken{}).Error; err != nil {
			log.Printf("[ERROR] Failed to delete refresh token: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error - Failed to delete refresh token",
			})
		}
	}

	// ✅ 5. Kosongkan cookie refresh_token di client
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  time.Now().Add(-time.Hour),
	})

	// ✅ 6. Balas sukses
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}
