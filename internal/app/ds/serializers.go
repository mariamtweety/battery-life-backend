package ds

import (
	"time"
)

// ScenarioListItemSerializer represents an item in the scenario list
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
	IsCreator           int        `json:"is_creator"` // 1 if current user created it, else 0
	IsLiked             int        `json:"is_liked"`   // 1 if current user liked it, else 0
}

// ScenarioDetailSerializer represents full scenario details including creator info
type ScenarioDetailSerializer struct {
	BatteryScenario
	LikesCount int64          `json:"likes_count"`
	IsCreator  int            `json:"is_creator"` // 1 if current user created it, else 0
	IsLiked    int            `json:"is_liked"`   // 1 if current user liked it, else 0
	Creator    BatteryAppUser `json:"creator"`
}

// UserScenariosSerializer demonstrates nested serialization: User with their scenarios
type UserScenariosSerializer struct {
	User      BatteryAppUser    `json:"user"`
	Scenarios []BatteryScenario `json:"scenarios"`
}

// ScenarioLikeResponse represents response for like toggle
type ScenarioLikeResponse struct {
	ScenarioID uint  `json:"scenario_id"`
	Liked      int   `json:"liked"`
	LikesCount int64 `json:"likes_count"`
}

// ScenarioLikeRequest represents incoming like request
type ScenarioLikeRequest struct {
	Like int `json:"like"` // 1 to like, 0 to unlike
}

// ScenarioPublishRequest represents incoming publish request
type ScenarioPublishRequest struct {
	ScenarioTitle       string  `json:"scenario_title"`
	ScenarioDescription string  `json:"scenario_description"`
	BatteryDrainMah     int     `json:"battery_drain_mah"`
	UsageDurationHours  float64 `json:"usage_duration_hours"`
}

// UserRegisterRequest represents incoming registration request
type UserRegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	IsModerator bool   `json:"is_moderator"`
}

// UserLoginRequest represents incoming login request
type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents authentication response stub
type AuthResponse struct {
	Message string          `json:"message"`
	Token   string          `json:"token,omitempty"`
	User    *BatteryAppUser `json:"user,omitempty"`
}

// SimpleMessageResponse represents generic informational response
type SimpleMessageResponse struct {
	Message string `json:"message"`
	ID      uint   `json:"id,omitempty"`
}
