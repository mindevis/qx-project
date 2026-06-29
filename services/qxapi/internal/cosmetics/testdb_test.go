package cosmetics

import (
	"testing"

	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	if err := db.AutoMigrate(&models.UserCosmetics{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
