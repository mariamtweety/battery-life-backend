package ds

import (
	"time"
)

type ScenarioListItemSerializer struct {
	ID                  uint       `json:"id"`
	ScenarioTitle       string     `json:"scenario_title"`
	ScenarioDescription string     `json:"scenario_description"`
	ScenarioStatus      string     `json:"scenario_status"`
	ImageURL            string     `json:"image_url"`
	VideoURL            string     `json:"video_url"`
	BatteryDrainMah     int        `json:"battery_drain_mah"`
	UsageDurationHours  float64    `json:"usage_duration_hours"`
	CreatedAt           time.Time  `json:"created_at"`
	FormationDate       *time.Time `json:"formation_date"`
	CreatedByUserID     uint       `json:"created_by_user_id"`
	LikesCount          int64      `json:"likes_count"`
	IsCreator           int        `json:"is_creator"`
	IsLiked             int        `json:"is_liked"`
}

type ScenarioDetailSerializer struct {
	BatteryScenario
	LikesCount int64          `json:"likes_count"`
	IsCreator  int            `json:"is_creator"`
	IsLiked    int            `json:"is_liked"`
	Creator    BatteryAppUser `json:"creator"`
}

type UserScenariosSerializer struct {
	User      BatteryAppUser    `json:"user"`
	Scenarios []BatteryScenario `json:"scenarios"`
}

type ScenarioLikeResponse struct {
	ScenarioID uint  `json:"scenario_id"`
	Liked      int   `json:"liked"`
	LikesCount int64 `json:"likes_count"`
}

type ScenarioLikeRequest struct {
	Like int `json:"like"`
}

type ScenarioPublishRequest struct {
	ScenarioTitle       string  `json:"scenario_title"`
	ScenarioDescription string  `json:"scenario_description"`
	BatteryDrainMah     int     `json:"battery_drain_mah"`
	UsageDurationHours  float64 `json:"usage_duration_hours"`
}

type UserRegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	IsModerator bool   `json:"is_moderator"`
}

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Message string          `json:"message"`
	Token   string          `json:"token,omitempty"`
	User    *BatteryAppUser `json:"user,omitempty"`
}

type SimpleMessageResponse struct {
	Message string `json:"message"`
	ID      uint   `json:"id,omitempty"`
}
