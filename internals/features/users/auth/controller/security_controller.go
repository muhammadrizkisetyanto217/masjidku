package controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	modelUser "masjidku/internals/features/users/user/models"
)

// 🔥 CHECK SECURITY ANSWER
func (ac *AuthController) CheckSecurityAnswer(c *fiber.Ctx) error {
	var input struct {
		Email  string `json:"email"`
		Answer string `json:"security_answer"`
	}

	// 📌 Parsing JSON input
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// 📌 Cek user berdasarkan email
	var user modelUser.UserModel
	if err := ac.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// 📌 Bandingkan security answer secara langsung
	if strings.TrimSpace(input.Answer) != strings.TrimSpace(user.SecurityAnswer) {
		return c.Status(400).JSON(fiber.Map{"error": "Incorrect security answer"})
	}

	// 📌 Response berhasil validasi
	return c.JSON(fiber.Map{
		"message": "Security answer correct",
		"email":   user.Email,
	})
}
