package database

import (
	"os"
	"strings"

	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

func ExecuteTriggers(db *gorm.DB) error {
	// Baca file SQL
	triggerSQL, err := os.ReadFile("database/migrations/triggers.sql")
	if err != nil {
		return err
	}

	// Split berdasarkan DELIMITER
	statements := strings.Split(string(triggerSQL), "DELIMITER")

	for _, block := range statements {
		if strings.TrimSpace(block) == "" {
			continue
		}

		// Eksekusi setiap statement dalam blok
		for _, stmt := range strings.Split(block, "//") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || stmt == ";" {
				continue
			}

			if err := db.Exec(stmt).Error; err != nil {
				utils.ErrorLogger.Printf("Error executing trigger: %v\nStatement: %s", err, stmt)
				continue
			}
			utils.InfoLogger.Printf("Successfully executed trigger statement")
		}
	}

	// Verifikasi trigger
	var triggers []struct {
		Trigger string
		Event   string
		Table   string
		Timing  string
	}

	db.Raw(`
        SELECT 
            TRIGGER_NAME as trigger_name,
            EVENT_MANIPULATION as event_type,
            EVENT_OBJECT_TABLE as table_name,
            ACTION_TIMING as timing
        FROM information_schema.triggers
        WHERE TRIGGER_SCHEMA = DATABASE()
    `).Scan(&triggers)

	for _, t := range triggers {
		utils.InfoLogger.Printf("Trigger verified: %s (%s %s on %s)",
			t.Trigger, t.Timing, t.Event, t.Table)
	}

	return nil
}
