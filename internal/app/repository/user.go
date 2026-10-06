package repository

import (
	"errors"
	"fmt"
	"time"

	"battery-life/internal/app/ds"

	"gorm.io/gorm"
)

// RegisterUser registers a new user in the system
func (r *Repository) RegisterUser(username, password string, isModerator bool) (*ds.BatteryAppUser, error) {
	var count int64
	err := r.db.Model(&ds.BatteryAppUser{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("пользователь с именем '%s' уже существует", username)
	}

	user := ds.BatteryAppUser{
		Username:     username,
		PasswordHash: password, // In lab 4 will be hashed with bcrypt
		IsModerator:  isModerator,
		CreatedAt:    time.Now(),
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	return &user, nil
}

// GetUserByID fetches user by ID
func (r *Repository) GetUserByID(id uint) (*ds.BatteryAppUser, error) {
	var user ds.BatteryAppUser
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserScenarios implements nested serialization: user and their scenarios
func (r *Repository) GetUserScenarios(userID uint) (*ds.UserScenariosSerializer, error) {
	var user ds.BatteryAppUser
	err := r.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("пользователь с id %d не найден", userID)
		}
		return nil, err
	}

	var scenarios []ds.BatteryScenario
	// Exclude deleted scenarios from client
	err = r.db.Where("created_by_user_id = ? AND scenario_status != 'deleted'", userID).
		Order("id ASC").
		Find(&scenarios).Error
	if err != nil {
		return nil, err
	}

	return &ds.UserScenariosSerializer{
		User:      user,
		Scenarios: scenarios,
	}, nil
}
