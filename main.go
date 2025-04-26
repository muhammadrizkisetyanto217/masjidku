package main

import (
	"log"
	"masjidku/internals/configs"
	"masjidku/internals/database"
	scheduler "masjidku/internals/features/users/auth/scheduler"
	routes "masjidku/internals/route"

	// "masjidku/internals/features/models"

	"github.com/gofiber/fiber/v2"
)

func main() {

	// ✅ Muat file .env dulu
	configs.LoadEnv()
	// Inisialisasi Fiber
	app := fiber.New()

	// Koneksi ke Supabase
	database.ConnectDB()

	// ✅ Jalankan scheduler harian
	scheduler.StartBlacklistCleanupScheduler(database.DB)

	// ✅ Panggil semua route dari folder routes
	routes.SetupRoutes(app, database.DB)

	// Start server
	log.Fatal(app.Listen(":3000"))
}
