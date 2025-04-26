package controller

import (
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	modelUser "masjidku/internals/features/users/user/models"
)

// 🔥 CHANGE PASSWORD (Menggunakan c.Locals dan Transaksi)
func (ac *AuthController) ChangePassword(c *fiber.Ctx) error {
	// 🆔 Ambil User ID dari middleware (sudah divalidasi di AuthMiddleware)
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized - Invalid token"})
	}

	// 📌 Parsing request body
	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// 📌 Validasi input kosong
	if input.OldPassword == "" || input.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Both old and new passwords are required"})
	}

	// 🚨 Cek apakah password baru sama dengan yang lama
	if input.OldPassword == input.NewPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be different from old password"})
	}

	// 🔍 Cari user di database
	var user modelUser.UserModel
	if err := ac.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// 🔑 Cek apakah password lama cocok
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Old password is incorrect"})
	}

	// 🔒 Hash password baru
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash new password"})
	}

	// 🔥 Update password menggunakan transaksi
	tx := ac.DB.Begin()
	if err := tx.Model(&user).Update("password", string(newHashedPassword)).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}
	tx.Commit()

	// 🎉 Beri response sukses
	return c.JSON(fiber.Map{"message": "Password changed successfully"})
}

// 🔥 RESET PASSWORD
func (ac *AuthController) ResetPassword(c *fiber.Ctx) error {
	var input struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
	}

	// 📌 Parsing JSON input
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// 📌 Cek user berdasarkan email kembali untuk memastikan
	var user modelUser.UserModel
	if err := ac.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// 📌 Hashing password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash new password"})
	}

	// 📌 Update password di database
	if err := ac.DB.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update password"})
	}

	// 📌 Response sukses reset password
	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}
