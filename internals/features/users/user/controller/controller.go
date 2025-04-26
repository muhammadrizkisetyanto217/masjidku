package controller

import "github.com/gofiber/fiber/v2"

func GetProfile(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Ini halaman profile user/teacher/staff/owner",
	})
}

func OwnerDashboard(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Dashboard hanya untuk owner",
	})
}
