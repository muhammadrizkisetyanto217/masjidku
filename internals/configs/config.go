package configs

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var (
	JWTSecret        string
	JWTRefreshSecret string
	DB               *gorm.DB
)

// =======================
// ENV LOADER
// =======================
func LoadEnv() {
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		if err := godotenv.Load(); err != nil {
			log.Println("⚠️ Tidak menemukan .env file, menggunakan ENV dari sistem")
		} else {
			log.Println("✅ .env file berhasil dimuat!")
		}
	} else {
		log.Println("🚀 Running in Railway, menggunakan ENV dari sistem")
	}

	JWTSecret = GetEnv("JWT_SECRET")
	JWTRefreshSecret = GetEnv("JWT_REFRESH_SECRET") // ⬅️ tambahkan ini

	if JWTSecret == "" {
		log.Println("❌ JWT_SECRET belum diset!")
	} else {
		log.Println("✅ JWT_SECRET berhasil dimuat.")
	}

	if JWTRefreshSecret == "" {
		log.Println("❌ JWT_REFRESH_SECRET belum diset!")
	} else {
		log.Println("✅ JWT_REFRESH_SECRET berhasil dimuat.")
	}

}

func GetEnv(key string, defaultValue ...string) string {
	value, exists := os.LookupEnv(key)
	if !exists && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return value
}

// =======================
// DATABASE CONNECTOR
// =======================
func InitDB() *gorm.DB {
	dsn := GetEnv("DB_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: NewGormLogger(), // pakai logger custom
	})
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke database: %v", err)
	}
	log.Println("✅ Database terkoneksi.")

	DB = db
	return db
}

// =======================
// GORM LOGGER CUSTOM
// =======================
type GormLogger struct {
	SlowThreshold time.Duration
	LogLevel      gormLogger.LogLevel
}

func NewGormLogger() gormLogger.Interface {
	return &GormLogger{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      gormLogger.Info,
	}
}

func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	l.LogLevel = level
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[INFO] "+msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[WARN] "+msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[ERROR] "+msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	file := utils.FileWithLineNum()

	if err != nil {
		log.Printf("[ERROR] %s | %v | %s | %d rows | %s", file, err, elapsed, rows, sql)
	} else if elapsed > l.SlowThreshold {
		log.Printf("[SLOW SQL] %s | %s | %d rows | %s", file, elapsed, rows, sql)
	} else {
		log.Printf("[QUERY] %s | %s | %d rows | %s", file, elapsed, rows, sql)
	}
}
