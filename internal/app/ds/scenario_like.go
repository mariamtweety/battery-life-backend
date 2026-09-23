package ds

import "time"

type ScenarioLike struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_user_scenario_like;column:user_id" json:"user_id"`
	ScenarioID uint      `gorm:"not null;uniqueIndex:idx_user_scenario_like;column:scenario_id" json:"scenario_id"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`

	User     PhoneUser       `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"user"`
	Scenario BatteryScenario `gorm:"foreignKey:ScenarioID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"scenario"`
}

func (ScenarioLike) TableName() string {
	return "scenario_likes"
}
