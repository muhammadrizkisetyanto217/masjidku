package auth

import (
	"errors"
	"log"

	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"masjidku/internals/configs"
	modelAuth "masjidku/internals/features/users/auth/models"

	"gorm.io/gorm"
)


// 🔥 Middleware untuk proteksi route
func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// 🚨 Skip middleware untuk Midtrans webhook
		if c.Path() == "/api/donations/notification" {
			log.Println("[INFO] Skip AuthMiddleware untuk webhook Midtrans")
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		log.Println("[DEBUG] Authorization Header:", authHeader)
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - No token provided"})
		}
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Invalid token format"})
		}
		tokenString := tokenParts[1]
		var existingToken modelAuth.TokenBlacklist
		err := db.Where("token = ?", tokenString).First(&existingToken).Error
		if err == nil {
			log.Println("[WARNING] Token ditemukan di blacklist, akses ditolak.")
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Token is blacklisted"})
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("[ERROR] Database error saat cek token blacklist:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
		}
		secretKey := configs.JWTSecret
		if secretKey == "" {
			log.Println("[ERROR] JWT_SECRET tidak ditemukan di environment")
			return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error - Missing JWT Secret"})
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil || !token.Valid {
			log.Println("[ERROR] Token tidak valid:", err)
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Invalid token"})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Println("[ERROR] Token claims tidak valid")
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Invalid token claims"})
		}
		exp, exists := claims["exp"].(float64)
		if !exists {
			log.Println("[ERROR] Token tidak memiliki exp")
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Token has no expiration"})
		}
		log.Println("[DEBUG] Token Claims:", claims)

		idStr, exists := claims["id"].(string)
		if !exists {
			log.Println("[ERROR] User ID not found in token claims")
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - No user ID in token"})
		}

		userID, err := uuid.Parse(idStr)
		if err != nil {
			log.Println("[ERROR] Failed to parse UUID from token:", err)
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Invalid user ID format"})
		}

		c.Locals("user_id", userID)
		log.Println("[SUCCESS] User ID stored in context:", userID)

		if role, ok := claims["role"].(string); ok {
			c.Locals("role", role)
		}
		if userName, ok := claims["user_name"].(string); ok {
			c.Locals("user_name", userName)
		}

		expTime := time.Unix(int64(exp), 0)
		log.Printf("[INFO] Token Expiration Time: %v", expTime)
		if time.Now().Unix() > int64(exp) {
			log.Println("[ERROR] Token sudah expired")
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized - Token expired"})
		}
		log.Println("[SUCCESS] Token valid, lanjutkan request")
		return c.Next()
	}
}
