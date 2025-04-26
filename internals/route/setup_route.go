package routes

import (
	
	userRoute "masjidku/internals/features/users/auth/route"
	authRoute "masjidku/internals/features/users/user/route"


	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

)

// Register routes
func SetupRoutes(app *fiber.App, db *gorm.DB) {

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Fiber & Supabase PostgreSQL connected successfully 🚀")
	})

	userRoute.AuthRoutes(app, db)
	authRoute.UserRoutes(app, db)

}
