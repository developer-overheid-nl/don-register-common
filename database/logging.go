package database

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ConfigureDefaultLogging structures GORM events emitted while opening and
// migrating a database.
func ConfigureDefaultLogging(logger *slog.Logger) {
	gormlogger.Default = newStructuredLogger(logger)
}

// ConfigureLogging sends runtime GORM events through the application logger.
func ConfigureLogging(db *gorm.DB, logger *slog.Logger) {
	db.Logger = newStructuredLogger(logger)
}

func newStructuredLogger(logger *slog.Logger) gormlogger.Interface {
	if logger == nil {
		logger = slog.Default()
	}

	return gormlogger.NewSlogLogger(
		logger.With("component", "database", "operation", "query"),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		},
	)
}
