package ds

type ScenarioListItemSerializer struct {
	ID                 uint    `json:"id"`
	ScenarioTitle      string  `json:"scenario_title"`
	ImageURL           string  `json:"image_url"`
	BatteryDrainMah    int     `json:"battery_drain_mah"`
	UsageDurationHours float64 `json:"usage_duration_hours"`
	LikesCount         int64   `json:"likes_count"`
}

type ScenarioFeedItemSerializer struct {
	ID                  uint    `json:"id"`
	ScenarioTitle       string  `json:"scenario_title"`
	ScenarioDescription string  `json:"scenario_description"`
	VideoURL            string  `json:"video_url"`
	BatteryDrainMah     int     `json:"battery_drain_mah"`
	UsageDurationHours  float64 `json:"usage_duration_hours"`
	LikesCount          int64   `json:"likes_count"`
	IsLiked             int     `json:"is_liked"`
}

type ScenarioDraftResponse struct {
	ID            uint   `json:"id"`
	ScenarioTitle string `json:"scenario_title"`
	ImageURL      string `json:"image_url,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
}

type ScenarioDraftDetailSerializer struct {
	ID                  uint    `json:"id"`
	ScenarioTitle       string  `json:"scenario_title"`
	ScenarioDescription string  `json:"scenario_description"`
	ImageURL            string  `json:"image_url,omitempty"`
	VideoURL            string  `json:"video_url,omitempty"`
	BatteryDrainMah     int     `json:"battery_drain_mah"`
	UsageDurationHours  float64 `json:"usage_duration_hours"`
	ScenarioStatus      string  `json:"scenario_status"`
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
