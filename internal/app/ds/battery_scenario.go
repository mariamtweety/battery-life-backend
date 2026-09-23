package ds

import "time"

type BatteryScenario struct {
	ID                  uint       `gorm:"primaryKey;column:id" json:"id"`
	ScenarioTitle       string     `gorm:"type:varchar(24);not null;column:scenario_title;check:length(trim(scenario_title)) >= 1 AND length(scenario_title) <= 24" json:"scenario_title"`
	ScenarioDescription string     `gorm:"type:varchar(1000);default:'';column:scenario_description;check:length(scenario_description) <= 1000" json:"scenario_description"`
	ScenarioStatus      string     `gorm:"type:varchar(20);not null;default:'draft';column:scenario_status" json:"scenario_status"`
	ImageURL            string     `gorm:"type:varchar(500);default:'';column:image_url" json:"image_url"`
	VideoURL            string     `gorm:"type:varchar(500);default:'';column:video_url" json:"video_url"`
	BatteryDrainMah     int        `gorm:"not null;default:0;column:battery_drain_mah;check:battery_drain_mah >= 0 AND battery_drain_mah <= 9999" json:"battery_drain_mah"`
	UsageDurationHours  float64    `gorm:"type:numeric(4,1);not null;default:0;column:usage_duration_hours" json:"usage_duration_hours"`
	CreatedAt           time.Time  `gorm:"not null;column:created_at" json:"created_at"`
	FormationDate       *time.Time `gorm:"column:formation_date" json:"formation_date"`
	CreatedByUserID     uint       `gorm:"not null;column:created_by_user_id" json:"created_by_user_id"`

	Creator PhoneUser `gorm:"foreignKey:CreatedByUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"creator"`
}

func (BatteryScenario) TableName() string {
	return "battery_scenarios"
}
