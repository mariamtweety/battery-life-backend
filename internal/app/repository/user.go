package repository

import (
	"errors"
	"fmt"
	"time"

	"battery-life/internal/app/ds"

	"gorm.io/gorm"
)

func (r *Repository) RegisterUser(username, passwordHash string, isModerator bool) (*ds.BatteryAppUser, error) {
	var count int64
	err := r.db.Model(&ds.BatteryAppUser{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: пользователь с именем '%s' уже существует", ErrConflict, username)
	}

	user := ds.BatteryAppUser{
		Username:     username,
		PasswordHash: passwordHash,
		IsModerator:  isModerator,
		CreatedAt:    time.Now(),
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	return &user, nil
}

// GetUserByUsername возвращает пользователя для аутентификации.
func (r *Repository) GetUserByUsername(username string) (*ds.BatteryAppUser, error) {
	var user ds.BatteryAppUser
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: пользователь с именем '%s' не найден", ErrNotFound, username)
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByID(id uint) (*ds.BatteryAppUser, error) {
	var user ds.BatteryAppUser
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: пользователь с id %d не найден", ErrNotFound, id)
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUserPasswordHash обновляет хэш пароля (например, при переходе с ЛР3 на bcrypt).
func (r *Repository) UpdateUserPasswordHash(id uint, passwordHash string) error {
	return r.db.Model(&ds.BatteryAppUser{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

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
