package route

import (
	controller "masjidku/internals/features/users/auth/controller"
	authMw "masjidku/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {
	authController := controller.NewAuthController(db)
	googleAuthController := controller.NewGoogleAuthController(db)

	auth := app.Group("/auth")
	auth.Post("/register", authController.Register)
	auth.Post("/login", authController.Login)
	auth.Post("/refresh-token", authController.RefreshToken)
	auth.Post("/forgot-password/check", authController.CheckSecurityAnswer)
	auth.Post("/forgot-password/reset", authController.ResetPassword)

	// Protected routes
	protectedRoutes := app.Group("/api/auth", authMw.AuthMiddleware(db))
	protectedRoutes.Post("/logout", authController.Logout)
	protectedRoutes.Post("/change-password", authController.ChangePassword)

	// Google auth
	auth.Get("/google", googleAuthController.GoogleLogin)
	auth.Get("/google/callback", googleAuthController.GoogleCallback)
}
