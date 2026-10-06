package singleton

import (
	"sync"
	"time"

	"battery-life/internal/app/ds"
)

// DefaultUserID is the fixed user ID for Lab 3
const DefaultUserID uint = 1

var (
	currentUserInstance *ds.BatteryAppUser
	userOnce            sync.Once
)

// GetCurrentUser returns the singleton instance of the fixed current user.
// Thread-safe initialization using sync.Once.
func GetCurrentUser() *ds.BatteryAppUser {
	userOnce.Do(func() {
		currentUserInstance = &ds.BatteryAppUser{
			ID:           DefaultUserID,
			Username:     "tester_alex",
			IsModerator:  false,
			CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			PasswordHash: "pass123",
		}
	})
	return currentUserInstance
}

// GetCurrentUserID returns the fixed current user ID from the singleton
func GetCurrentUserID() uint {
	return GetCurrentUser().ID
}
