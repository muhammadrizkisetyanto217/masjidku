package controller

import (
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"masjidku/internals/features/users/user/models"

	"gorm.io/gorm"
)

// * Kita membuat sebuah struct bernama UserController, yang memiliki satu property bernama DB. (Property adalah variabel yang terdapat dalam sebuah struct).
// & Property DB ini adalah pointer ke objek database (gorm.DB), yang akan digunakan untuk mengakses database.
type UserController struct {
	DB *gorm.DB
}

//^ Bayangkan UserController ini seperti seorang kasir toko.
// 1. Agar bisa bekerja, kasir butuh akses ke database toko (misalnya, daftar barang dan harga).
// 2. Dalam kode ini, DB adalah akses ke database yang diberikan ke kasir (UserController).
// 3. Tanpa DB, kasir tidak bisa mencari barang, menambahkan transaksi, dll.

// *  Fungsi NewUserController adalah "constructor"
// Constructor ini digunakan untuk membuat objek UserController dengan database yang bisa disesuaikan.
func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// 1. Saat Anda mempekerjakan kasir baru (UserController), Anda harus memberi mereka akses ke database toko (DB).
// 2. NewUserController(db) adalah cara memberi kasir akses ke database saat mereka mulai bekerja.

// GET all users
func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []models.UserModel
	if err := uc.DB.Find(&users).Error; err != nil {
		log.Println("[ERROR] Failed to fetch users:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve users"})
	}

	log.Printf("[SUCCESS] Retrieved %d users\n", len(users))
	return c.JSON(fiber.Map{
		"message": "Users fetched successfully",
		"total":   len(users),
		"data":    users,
	})
}

// GET user by ID
func (uc *UserController) GetProfile(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var user models.UserModel
	if err := uc.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(fiber.Map{
		"message": "User profile fetched successfully",
		"data":    user,
	})
}

// POST create new user(s)
func (uc *UserController) CreateUser(c *fiber.Ctx) error {
	var singleUser models.UserModel
	var multipleUsers []models.UserModel

	// Coba parse sebagai array terlebih dahulu
	if err := c.BodyParser(&multipleUsers); err == nil && len(multipleUsers) > 0 {
		if err := uc.DB.Create(&multipleUsers).Error; err != nil {
			log.Println("[ERROR] Failed to create multiple users:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create multiple users"})
		}
		return c.Status(201).JSON(fiber.Map{
			"message": "Users created successfully",
			"data":    multipleUsers,
		})
	}

	// Jika gagal diparse sebagai array, parse sebagai satu user
	if err := c.BodyParser(&singleUser); err != nil {
		log.Println("[ERROR] Invalid input format:", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input format"})
	}

	if err := uc.DB.Create(&singleUser).Error; err != nil {
		log.Println("[ERROR] Failed to create user:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create user"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "User created successfully",
		"data":    singleUser,
	})
}

// PUT update user by ID
type UpdateUserInput struct {
	UserName     string  `json:"user_name" validate:"required,min=3,max=50"`
	Email        string  `json:"email" validate:"required,email"`
	DonationName *string `json:"donation_name"`
	OriginalName *string `json:"original_name"`
}

// UpdateProfile - Update user dari token
func (uc *UserController) UpdateProfile(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userIDStr, ok := userIDRaw.(string)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid user ID in token"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid UUID format"})
	}

	var user models.UserModel
	if err := uc.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	var input UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validasi input manual atau pakai validator.v10
	validate := validator.New()
	if err := validate.Struct(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Update field yang diizinkan
	user.UserName = input.UserName
	user.Email = input.Email
	user.DonationName = input.DonationName
	user.OriginalName = input.OriginalName

	if err := uc.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update user"})
	}

	return c.JSON(fiber.Map{
		"message": "User updated successfully",
		"data": fiber.Map{
			"id":            user.ID,
			"user_name":     user.UserName,
			"email":         user.Email,
			"donation_name": user.DonationName,
			"original_name": user.OriginalName,
			"updated_at":    user.UpdatedAt,
		},
	})
}

// DELETE user by ID
func (uc *UserController) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := uc.DB.Delete(&models.UserModel{}, id).Error; err != nil {
		log.Println("[ERROR] Failed to delete user:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete user"})
	}

	log.Printf("[SUCCESS] User with ID %s deleted\n", id)
	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
