package repository

import (
	"errors"
	"fmt"
	"time"

	"battery-life/internal/app/ds"

	"gorm.io/gorm"
)

// GetPublishedScenarios возвращает все опубликованные сценарии через ORM
func (r *Repository) GetPublishedScenarios() ([]ds.BatteryScenario, error) {
	var scenarios []ds.BatteryScenario
	err := r.db.Where("scenario_status = ?", "published").Order("id ASC").Find(&scenarios).Error
	if err != nil {
		return nil, err
	}
	return scenarios, nil
}

// GetFirstPublishedScenario возвращает первый опубликованный сценарий для ленты
func (r *Repository) GetFirstPublishedScenario() (*ds.BatteryScenario, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("scenario_status = ?", "published").Order("id ASC").First(&scenario).Error
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

// GetScenarioByID возвращает опубликованный сценарий по ID. Удаленные и черновики не возвращаются.
func (r *Repository) GetScenarioByID(id uint) (*ds.BatteryScenario, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("id = ? AND scenario_status = ?", id, "published").First(&scenario).Error
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

// GetNextPublishedScenario возвращает следующий опубликованный сценарий после currentID
func (r *Repository) GetNextPublishedScenario(currentID uint) (*ds.BatteryScenario, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("id > ? AND scenario_status = ?", currentID, "published").Order("id ASC").First(&scenario).Error
	if err == nil {
		return &scenario, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.GetFirstPublishedScenario()
	}
	return nil, err
}

// GetDraftScenarioByUserID возвращает черновик пользователя через ORM
func (r *Repository) GetDraftScenarioByUserID(userID uint) (*ds.BatteryScenario, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("created_by_user_id = ? AND scenario_status = ?", userID, "draft").First(&scenario).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &scenario, nil
}

// CreateDraftScenario создает черновик сценария через ORM
func (r *Repository) CreateDraftScenario(scenario *ds.BatteryScenario) error {
	scenario.ScenarioStatus = "draft"
	scenario.CreatedAt = time.Now()
	scenario.FormationDate = nil
	return r.db.Create(scenario).Error
}

// PublishScenario публикует сценарий (смена статуса на published) через ORM
func (r *Repository) PublishScenario(id uint, description string, drainMah int, durationHours float64) error {
	now := time.Now()
	updates := map[string]interface{}{
		"scenario_description": description,
		"battery_drain_mah":    drainMah,
		"usage_duration_hours": durationHours,
		"scenario_status":      "published",
		"formation_date":       now,
	}
	result := r.db.Model(&ds.BatteryScenario{}).Where("id = ? AND scenario_status = ?", id, "draft").Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("черновик с id %d не найден для публикации", id)
	}
	return nil
}

// GetLikesCount возвращает количество лайков для сценария из m-m таблицы через ORM
func (r *Repository) GetLikesCount(scenarioID uint) int64 {
	var count int64
	err := r.db.Model(&ds.ScenarioLike{}).Where("scenario_id = ?", scenarioID).Count(&count).Error
	if err != nil {
		return 0
	}
	return count
}

// GetScenariosWithLikes возвращает список опубликованных сценариев с лайками и фильтрацией
func (r *Repository) GetScenariosWithLikes(nameFilter string, maxHours float64) ([]ScenarioWithLikes, error) {
	query := r.db.Model(&ds.BatteryScenario{}).Where("scenario_status = ?", "published")

	if nameFilter != "" {
		query = query.Where("scenario_title ILIKE ?", "%"+nameFilter+"%")
	}
	if maxHours > 0 {
		query = query.Where("usage_duration_hours <= ?", maxHours)
	}

	var scenarios []ds.BatteryScenario
	if err := query.Order("id ASC").Find(&scenarios).Error; err != nil {
		return nil, err
	}

	result := make([]ScenarioWithLikes, 0, len(scenarios))
	for _, s := range scenarios {
		likesCount := r.GetLikesCount(s.ID)
		result = append(result, ScenarioWithLikes{
			BatteryScenario: s,
			LikesCount:      likesCount,
		})
	}

	return result, nil
}

// DeleteScenarioSQL выполняет логическое удаление сценария (статус меняется на 'deleted') с помощью SQL UPDATE БЕЗ ORM
func (r *Repository) DeleteScenarioSQL(scenarioID uint) error {
	rawSQL := "UPDATE battery_scenarios SET scenario_status = $1 WHERE id = $2"
	result := r.db.Exec(rawSQL, "deleted", scenarioID)
	if result.Error != nil {
		return fmt.Errorf("ошибка SQL UPDATE при удалении сценария: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("сценарий с id %d не найден для удаления", scenarioID)
	}
	return nil
}
