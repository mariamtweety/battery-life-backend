package singleton

import (
	"battery-life/internal/app/ds"
)

const CurrentUserID uint = 1

func GetCurrentUserID() uint {
	return CurrentUserID
}

func GetCurrentUser() *ds.BatteryAppUser {
	return &ds.BatteryAppUser{
		ID:          CurrentUserID,
		Username:    "tester_alex",
		IsModerator: false,
	}
}
