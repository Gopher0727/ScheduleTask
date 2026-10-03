// Package testutil supplies isolated schemas in an explicitly configured test DB.
package testutil

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Postgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("SCHEDULETASK_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SCHEDULETASK_TEST_POSTGRES_DSN to a disposable PostgreSQL database")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		a, _ := admin.DB()
		a.Close()
	})
	return db
}
