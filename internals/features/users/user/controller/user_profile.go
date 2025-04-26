package controller

import (
	"log"
	"strconv"

	"masjidku/internals/features/users/user/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersProfileController struct {
	DB *gorm.DB
}

func NewUsersProfileController(db *gorm.DB) *UsersProfileController {
	return &UsersProfileController{DB: db}
}

func (upc *UsersProfileController) GetProfiles(c *fiber.Ctx) error {
	log.Println("Fetching all user profiles")
	var profiles []models.UsersProfileModel
	if err := upc.DB.Find(&profiles).Error; err != nil {
		log.Println("Error fetching user profiles:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user profiles"})
	}
	return c.JSON(profiles)
}

func (upc *UsersProfileController) GetProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Println("Fetching user profile with ID:", id)
	var profile models.UsersProfileModel
	if err := upc.DB.First(&profile, id).Error; err != nil {
		log.Println("User profile not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User profile not found"})
	}
	return c.JSON(profile)
}

func (upc *UsersProfileController) CreateProfile(c *fiber.Ctx) error {
	log.Println("Creating or updating user profile")

	// Parse request body
	var input models.UsersProfileModel
	if err := c.BodyParser(&input); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validasi wajib user_id
	if input.UserID == uuid.Nil {
		log.Println("Missing user_id")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}

	// Cari apakah user_id sudah ada di database
	var existingProfile models.UsersProfileModel
	result := upc.DB.Where("user_id = ?", input.UserID).First(&existingProfile)

	if result.RowsAffected > 0 {
		// Update profil yang sudah ada
		if err := upc.DB.Model(&existingProfile).Updates(input).Error; err != nil {
			log.Println("Error updating user profile:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update user profile"})
		}
		log.Println("User profile updated:", input.UserID)
		return c.JSON(existingProfile)
	}

	// Buat profil baru jika user_id belum ada
	if err := upc.DB.Create(&input).Error; err != nil {
		log.Println("Error creating user profile:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user profile"})
	}

	log.Println("User profile created:", input.UserID)
	return c.Status(fiber.StatusCreated).JSON(input)
}

func (upc *UsersProfileController) UpdateProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Println("Updating user profile with ID:", id)
	var profile models.UsersProfileModel

	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Println("Invalid ID format:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
	}

	if err := upc.DB.First(&profile, idInt).Error; err != nil {
		log.Println("User profile not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User profile not found"})
	}

	if err := c.BodyParser(&profile); err != nil {
		log.Println("Invalid request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	profile.ID = uint(idInt) // Pastikan ID tidak berubah dari request body

	if err := upc.DB.Save(&profile).Error; err != nil {
		log.Println("Error updating user profile:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update user profile"})
	}
	return c.JSON(profile)
}

func (upc *UsersProfileController) DeleteProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Println("Deleting user profile with ID:", id)

	if err := upc.DB.Delete(&models.UsersProfileModel{}, id).Error; err != nil {
		log.Println("Error deleting user profile:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user profile"})
	}
	return c.JSON(fiber.Map{"message": "User profile deleted successfully"})
}
