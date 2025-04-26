package route

import (
	authController "masjidku/internals/middlewares/auth"
	userController "masjidku/internals/features/users/user/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupRoutes mengatur routing untuk user & user profile
func UserRoutes(app *fiber.App, db *gorm.DB) {

	// ✅ Group dengan Middleware Auth
	api := app.Group("/api", authController.AuthMiddleware(db))

	// 🔹 Users
	userCtrl := userController.NewUserController(db)
	userRoutes := api.Group("/users")
	userRoutes.Get("/", userCtrl.GetUsers)
	userRoutes.Get("/profile", userCtrl.GetProfile)
	userRoutes.Put("/profile", userCtrl.UpdateProfile)
	userRoutes.Delete("/:id", userCtrl.DeleteUser)

	// 🔹 Users Profile
	userProfileCtrl := userController.NewUsersProfileController(db)
	usersProfileRoutes := api.Group("/users-profiles")
	usersProfileRoutes.Get("/", userProfileCtrl.GetProfiles)
	usersProfileRoutes.Get("/:id", userProfileCtrl.GetProfile)
	usersProfileRoutes.Post("/", userProfileCtrl.CreateProfile)
	usersProfileRoutes.Put("/:id", userProfileCtrl.UpdateProfile)
	usersProfileRoutes.Delete("/:id", userProfileCtrl.DeleteProfile)
}
