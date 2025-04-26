package scheduler

import (
	"log"
	"time"

	"masjidku/internals/features/users/auth/models"

	"gorm.io/gorm"
)

func StartBlacklistCleanupScheduler(db *gorm.DB) {
	go func() {
		for {
			log.Println("[CLEANUP] Menjalankan pembersihan token_blacklist...")
			result := db.Where("expired_at < ?", time.Now()).Delete(&models.TokenBlacklist{})
			if result.Error != nil {
				log.Printf("[CLEANUP ERROR] %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("[CLEANUP] %d token kadaluarsa dihapus", result.RowsAffected)
			} else {
				log.Println("[CLEANUP] Tidak ada token kadaluarsa ditemukan")
			}

			time.Sleep(24 * time.Hour)
		}
	}()
}
