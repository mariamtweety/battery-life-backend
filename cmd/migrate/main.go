package main

import (
	"log"
	"time"

	"battery-life/internal/app/ds"
	"battery-life/internal/app/dsn"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	dsnStr := dsn.FromEnv()
	log.Printf("Connecting to Postgres with DSN: %s", dsnStr)

	db, err := gorm.Open(postgres.Open(dsnStr), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	renameLegacyTables(db)

	log.Println("Running AutoMigrate for 3 domain models...")
	err = db.AutoMigrate(
		&ds.BatteryAppUser{},
		&ds.BatteryScenario{},
		&ds.BatteryScenarioLike{},
	)
	if err != nil {
		log.Fatalf("cant migrate db: %v", err)
	}

	err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uq_user_single_draft ON battery_scenarios (created_by_user_id) WHERE scenario_status = 'draft';").Error
	if err != nil {
		log.Fatalf("cant create draft unique index: %v", err)
	}

	_ = db.Exec("ALTER TABLE battery_scenarios DROP CONSTRAINT IF EXISTS chk_battery_drain_mah;")
	err = db.Exec("ALTER TABLE battery_scenarios ADD CONSTRAINT chk_battery_drain_mah CHECK (battery_drain_mah >= 0 AND battery_drain_mah <= 9999);").Error
	if err != nil {
		log.Fatalf("cant create battery_drain_mah check constraint: %v", err)
	}

	_ = db.Exec("ALTER TABLE battery_scenarios DROP CONSTRAINT IF EXISTS chk_scenario_title_len;")
	err = db.Exec("ALTER TABLE battery_scenarios ADD CONSTRAINT chk_scenario_title_len CHECK (length(trim(scenario_title)) >= 1 AND length(scenario_title) <= 24);").Error
	if err != nil {
		log.Fatalf("cant create chk_scenario_title_len check constraint: %v", err)
	}

	_ = db.Exec("ALTER TABLE battery_scenarios DROP CONSTRAINT IF EXISTS chk_battery_scenarios_scenario_description;")
	_ = db.Exec("ALTER TABLE battery_scenarios DROP CONSTRAINT IF EXISTS chk_scenario_description_len;")
	err = db.Exec("ALTER TABLE battery_scenarios ADD CONSTRAINT chk_scenario_description_len CHECK (length(scenario_description) <= 1000);").Error
	if err != nil {
		log.Fatalf("cant create chk_scenario_description_len check constraint: %v", err)
	}

	seedData(db)
	log.Println("Database migration and initial seeding completed successfully!")
}

func renameLegacyTables(db *gorm.DB) {
	if db.Migrator().HasTable("phone_users") && !db.Migrator().HasTable("battery_app_users") {
		if err := db.Migrator().RenameTable("phone_users", "battery_app_users"); err != nil {
			log.Fatalf("cant rename phone_users table: %v", err)
		}
		if err := db.Exec("ALTER SEQUENCE IF EXISTS phone_users_id_seq RENAME TO battery_app_users_id_seq").Error; err != nil {
			log.Fatalf("cant rename phone_users sequence: %v", err)
		}
	}

	if db.Migrator().HasTable("scenario_likes") && !db.Migrator().HasTable("battery_scenario_likes") {
		if err := db.Migrator().RenameTable("scenario_likes", "battery_scenario_likes"); err != nil {
			log.Fatalf("cant rename scenario_likes table: %v", err)
		}
		if err := db.Exec("ALTER SEQUENCE IF EXISTS scenario_likes_id_seq RENAME TO battery_scenario_likes_id_seq").Error; err != nil {
			log.Fatalf("cant rename scenario_likes sequence: %v", err)
		}
	}
}

func seedData(db *gorm.DB) {
	var usersCount int64
	db.Model(&ds.BatteryAppUser{}).Count(&usersCount)
	if usersCount == 0 {
		log.Println("Seeding battery_app_users...")
		users := []ds.BatteryAppUser{
			{
				ID:           1,
				Username:     "tester_alex",
				PasswordHash: "pass123",
				IsModerator:  false,
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				Username:     "admin_battery",
				PasswordHash: "admin123",
				IsModerator:  true,
				CreatedAt:    time.Now(),
			},
			{
				ID:           3,
				Username:     "tester_elena",
				PasswordHash: "pass456",
				IsModerator:  false,
				CreatedAt:    time.Now(),
			},
		}
		for _, u := range users {
			db.Create(&u)
		}
	}

	var scenariosCount int64
	db.Model(&ds.BatteryScenario{}).Count(&scenariosCount)
	if scenariosCount == 0 {
		log.Println("Seeding battery_scenarios...")
		now := time.Now()
		scenarios := []ds.BatteryScenario{
			{
				ID:                  1,
				ScenarioTitle:       "Ретро гейминг",
				ScenarioDescription: "Приятно проведите время за игрой детства, не переживая за автономность телефона",
				ScenarioStatus:      "published",
				ImageURL:            "http://localhost:9002/scenarios/gaming.png",
				VideoURL:            "http://localhost:9002/scenarios/gaming.mp4",
				BatteryDrainMah:     1000,
				UsageDurationHours:  5.0,
				CreatedAt:           now,
				FormationDate:       &now,
				CreatedByUserID:     1,
			},
			{
				ID:                  2,
				ScenarioTitle:       "Серфинг",
				ScenarioDescription: "Просматривайте любимые сайты без ограничений (кроме ограничений от РКН)",
				ScenarioStatus:      "published",
				ImageURL:            "http://localhost:9002/scenarios/scrolling.png",
				VideoURL:            "http://localhost:9002/scenarios/scrolling.mp4",
				BatteryDrainMah:     625,
				UsageDurationHours:  8.0,
				CreatedAt:           now,
				FormationDate:       &now,
				CreatedByUserID:     1,
			},
			{
				ID:                  3,
				ScenarioTitle:       "Просмотр кино",
				ScenarioDescription: "Наслаждайтесь фильмами в дороге, на паре, дома, на работе, да и вообще везде",
				ScenarioStatus:      "published",
				ImageURL:            "http://localhost:9002/scenarios/films.png",
				VideoURL:            "http://localhost:9002/scenarios/films.mp4",
				BatteryDrainMah:     700,
				UsageDurationHours:  6.0,
				CreatedAt:           now,
				FormationDate:       &now,
				CreatedByUserID:     1,
			},
			{
				ID:                  4,
				ScenarioTitle:       "Чтение",
				ScenarioDescription: "Читайте книги весь день, а еще лучше, грокайте машинное обучение - это весело!!",
				ScenarioStatus:      "published",
				ImageURL:            "http://localhost:9002/scenarios/reading.png",
				VideoURL:            "http://localhost:9002/scenarios/reading.mp4",
				BatteryDrainMah:     427,
				UsageDurationHours:  15.0,
				CreatedAt:           now,
				FormationDate:       &now,
				CreatedByUserID:     1,
			},
			{
				ID:                  5,
				ScenarioTitle:       "Фотосъемка",
				ScenarioDescription: "Снимайте фото без страха разрядиться, вместо этого переживайте за кадр!",
				ScenarioStatus:      "published",
				ImageURL:            "http://localhost:9002/scenarios/photography.png",
				VideoURL:            "http://localhost:9002/scenarios/photography.mp4",
				BatteryDrainMah:     810,
				UsageDurationHours:  10.0,
				CreatedAt:           now,
				FormationDate:       &now,
				CreatedByUserID:     1,
			},
			{
				ID:                  6,
				ScenarioTitle:       "Общение",
				ScenarioDescription: "Общайтесь сколько вам захочется и не думайте о зарядке аккумулятора",
				ScenarioStatus:      "deleted",
				ImageURL:            "http://localhost:9002/scenarios/chatting.png",
				VideoURL:            "http://localhost:9002/scenarios/chatting.mp4",
				BatteryDrainMah:     786,
				UsageDurationHours:  7.0,
				CreatedAt:           now,
				FormationDate:       &now,
				CreatedByUserID:     1,
			},
			{
				ID:                  7,
				ScenarioTitle:       "Слушать музыку",
				ScenarioDescription: "",
				ScenarioStatus:      "draft",
				ImageURL:            "",
				VideoURL:            "",
				BatteryDrainMah:     0,
				UsageDurationHours:  0,
				CreatedAt:           now,
				FormationDate:       nil,
				CreatedByUserID:     1,
			},
		}
		for _, s := range scenarios {
			db.Create(&s)
		}

		db.Exec("SELECT setval('battery_scenarios_id_seq', (SELECT MAX(id) FROM battery_scenarios));")
		db.Exec("SELECT setval('battery_app_users_id_seq', (SELECT MAX(id) FROM battery_app_users));")
	}

	var likesCount int64
	db.Model(&ds.BatteryScenarioLike{}).Count(&likesCount)
	if likesCount == 0 {
		log.Println("Seeding battery_scenario_likes...")
		likes := []ds.BatteryScenarioLike{
			{UserID: 1, ScenarioID: 1, CreatedAt: time.Now()},
			{UserID: 2, ScenarioID: 1, CreatedAt: time.Now()},
			{UserID: 1, ScenarioID: 2, CreatedAt: time.Now()},
			{UserID: 2, ScenarioID: 2, CreatedAt: time.Now()},
			{UserID: 3, ScenarioID: 2, CreatedAt: time.Now()},
			{UserID: 2, ScenarioID: 3, CreatedAt: time.Now()},
			{UserID: 3, ScenarioID: 4, CreatedAt: time.Now()},
		}
		for _, l := range likes {
			db.Create(&l)
		}
	}
}
