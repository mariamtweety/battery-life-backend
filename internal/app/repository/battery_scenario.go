package repository

import (
	"errors"
	"fmt"
	"time"

	"battery-life/internal/app/ds"

	"gorm.io/gorm"
)

func (r *Repository) GetPublishedScenarios() ([]ds.BatteryScenario, error) {
	var scenarios []ds.BatteryScenario
	err := r.db.Where("scenario_status = ?", "published").Order("id ASC").Find(&scenarios).Error
	if err != nil {
		return nil, err
	}
	return scenarios, nil
}

func (r *Repository) GetFirstPublishedScenario() (*ds.BatteryScenario, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("scenario_status = ?", "published").Order("id ASC").First(&scenario).Error
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

func (r *Repository) GetScenarioByID(id uint) (*ds.BatteryScenario, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("id = ? AND scenario_status = ?", id, "published").First(&scenario).Error
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

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

func (r *Repository) CreateDraftScenario(scenario *ds.BatteryScenario) error {
	scenario.ScenarioStatus = "draft"
	scenario.CreatedAt = time.Now()
	scenario.FormationDate = nil
	return r.db.Create(scenario).Error
}

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

func (r *Repository) GetLikesCount(scenarioID uint) int64 {
	var count int64
	err := r.db.Model(&ds.BatteryScenarioLike{}).Where("scenario_id = ?", scenarioID).Count(&count).Error
	if err != nil {
		return 0
	}
	return count
}

func (r *Repository) IsScenarioLikedByUser(userID, scenarioID uint) int {
	var count int64
	err := r.db.Model(&ds.BatteryScenarioLike{}).Where("user_id = ? AND scenario_id = ?", userID, scenarioID).Count(&count).Error
	if err != nil || count == 0 {
		return 0
	}
	return 1
}

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

func (r *Repository) MarkScenarioDeleted(scenarioID uint) error {
	query := "UPDATE battery_scenarios SET scenario_status = $1 WHERE id = $2 RETURNING id, scenario_title, scenario_status"

	rows, err := r.db.Raw(query, "deleted", scenarioID).Rows()
	if err != nil {
		return fmt.Errorf("ошибка выполнения SQL запроса через курсор rows: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("ошибка курсора rows: %w", err)
		}
		return fmt.Errorf("сценарий с id %d не найден для удаления", scenarioID)
	}

	var (
		id     uint
		title  string
		status string
	)

	if err := rows.Scan(&id, &title, &status); err != nil {
		return fmt.Errorf("ошибка сканирования данных из курсора rows: %w", err)
	}

	return nil
}

func (r *Repository) GetPublishedScenariosAPI(currentUserID uint, filterTitle string, maxHours float64) ([]ds.ScenarioListItemSerializer, error) {
	query := r.db.Model(&ds.BatteryScenario{}).Where("scenario_status = ?", "published")

	if filterTitle != "" {
		query = query.Where("scenario_title ILIKE ?", "%"+filterTitle+"%")
	}
	if maxHours > 0 {
		query = query.Where("usage_duration_hours <= ?", maxHours)
	}

	var scenarios []ds.BatteryScenario
	if err := query.Order("id ASC").Find(&scenarios).Error; err != nil {
		return nil, err
	}

	items := make([]ds.ScenarioListItemSerializer, 0, len(scenarios))
	for _, s := range scenarios {
		likesCount := r.GetLikesCount(s.ID)

		items = append(items, ds.ScenarioListItemSerializer{
			ID:                 s.ID,
			ScenarioTitle:      s.ScenarioTitle,
			ImageURL:           s.ImageURL,
			BatteryDrainMah:    s.BatteryDrainMah,
			UsageDurationHours: s.UsageDurationHours,
			LikesCount:         likesCount,
		})
	}

	return items, nil
}

func (r *Repository) GetFeedScenarioAPI(currentUserID uint, id *uint, next bool) (*ds.ScenarioFeedItemSerializer, error) {
	var scenario ds.BatteryScenario
	var err error

	if id == nil {
		err = r.db.Where("scenario_status = ?", "published").Order("id ASC").First(&scenario).Error
	} else if next {
		err = r.db.Where("id > ? AND scenario_status = ?", *id, "published").Order("id ASC").First(&scenario).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = r.db.Where("scenario_status = ?", "published").Order("id ASC").First(&scenario).Error
		}
	} else {
		err = r.db.Where("id = ? AND scenario_status = ?", *id, "published").First(&scenario).Error
	}

	if err != nil {
		return nil, err
	}

	isLiked := r.IsScenarioLikedByUser(currentUserID, scenario.ID)
	likesCount := r.GetLikesCount(scenario.ID)

	return &ds.ScenarioFeedItemSerializer{
		ID:                  scenario.ID,
		ScenarioTitle:       scenario.ScenarioTitle,
		ScenarioDescription: scenario.ScenarioDescription,
		VideoURL:            scenario.VideoURL,
		BatteryDrainMah:     scenario.BatteryDrainMah,
		UsageDurationHours:  scenario.UsageDurationHours,
		LikesCount:          likesCount,
		IsLiked:             isLiked,
	}, nil
}

func (r *Repository) GetDraftScenarioAPI(currentUserID uint) (*ds.ScenarioDraftDetailSerializer, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("created_by_user_id = ? AND scenario_status = ?", currentUserID, "draft").First(&scenario).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &ds.ScenarioDraftDetailSerializer{
		ID:                  scenario.ID,
		ScenarioTitle:       scenario.ScenarioTitle,
		ScenarioDescription: scenario.ScenarioDescription,
		ImageURL:            scenario.ImageURL,
		VideoURL:            scenario.VideoURL,
		BatteryDrainMah:     scenario.BatteryDrainMah,
		UsageDurationHours:  scenario.UsageDurationHours,
		ScenarioStatus:      scenario.ScenarioStatus,
	}, nil
}

func (r *Repository) CreateOrUpdateDraftScenario(scenario *ds.BatteryScenario) (*ds.BatteryScenario, error) {
	var existing ds.BatteryScenario
	err := r.db.Where("created_by_user_id = ? AND scenario_status = ?", scenario.CreatedByUserID, "draft").First(&existing).Error
	if err == nil {
		existing.ScenarioTitle = scenario.ScenarioTitle
		if scenario.ScenarioDescription != "" {
			existing.ScenarioDescription = scenario.ScenarioDescription
		}
		if scenario.BatteryDrainMah > 0 {
			existing.BatteryDrainMah = scenario.BatteryDrainMah
		}
		if scenario.UsageDurationHours > 0 {
			existing.UsageDurationHours = scenario.UsageDurationHours
		}
		if scenario.ImageURL != "" {
			existing.ImageURL = scenario.ImageURL
		}
		if scenario.VideoURL != "" {
			existing.VideoURL = scenario.VideoURL
		}
		if err := r.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	scenario.ScenarioStatus = "draft"
	scenario.CreatedAt = time.Now()
	scenario.FormationDate = nil
	if err := r.db.Create(scenario).Error; err != nil {
		return nil, err
	}
	return scenario, nil
}

func (r *Repository) UpdateScenarioMedia(scenarioID uint, imageURL, videoURL string) error {
	updates := map[string]interface{}{}
	if imageURL != "" {
		updates["image_url"] = imageURL
	}
	if videoURL != "" {
		updates["video_url"] = videoURL
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&ds.BatteryScenario{}).Where("id = ?", scenarioID).Updates(updates).Error
}

func (r *Repository) PublishScenarioAPI(scenarioID uint, currentUserID uint, req ds.ScenarioPublishRequest) (*ds.ScenarioDraftDetailSerializer, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("id = ?", scenarioID).First(&scenario).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("сценарий с id %d не найден", scenarioID)
		}
		return nil, err
	}

	if scenario.CreatedByUserID != currentUserID {
		return nil, fmt.Errorf("нет прав: вы не являетесь создателем этого сценария")
	}

	if scenario.ScenarioStatus == "published" {
		return nil, fmt.Errorf("сценарий уже опубликован")
	}
	if scenario.ScenarioStatus == "deleted" {
		return nil, fmt.Errorf("нельзя опубликовать удаленный сценарий")
	}
	if scenario.ScenarioStatus != "draft" {
		return nil, fmt.Errorf("сценарий не в статусе черновика (текущий статус: %s)", scenario.ScenarioStatus)
	}

	now := time.Now()
	scenario.ScenarioDescription = req.ScenarioDescription
	scenario.BatteryDrainMah = req.BatteryDrainMah
	scenario.UsageDurationHours = req.UsageDurationHours
	scenario.ScenarioStatus = "published"
	scenario.FormationDate = &now
	if req.ScenarioTitle != "" {
		scenario.ScenarioTitle = req.ScenarioTitle
	}

	if err := r.db.Save(&scenario).Error; err != nil {
		return nil, err
	}

	return &ds.ScenarioDraftDetailSerializer{
		ID:                  scenario.ID,
		ScenarioTitle:       scenario.ScenarioTitle,
		ScenarioDescription: scenario.ScenarioDescription,
		ImageURL:            scenario.ImageURL,
		VideoURL:            scenario.VideoURL,
		BatteryDrainMah:     scenario.BatteryDrainMah,
		UsageDurationHours:  scenario.UsageDurationHours,
		ScenarioStatus:      scenario.ScenarioStatus,
	}, nil
}

func (r *Repository) SoftDeleteScenarioAPI(scenarioID uint, currentUserID uint) error {
	var scenario ds.BatteryScenario
	err := r.db.Where("id = ?", scenarioID).First(&scenario).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("сценарий с id %d не найден", scenarioID)
		}
		return err
	}

	if scenario.CreatedByUserID != currentUserID {
		return fmt.Errorf("нет прав: только создатель может удалить свой сценарий")
	}

	if scenario.ScenarioStatus == "deleted" {
		return fmt.Errorf("сценарий с id %d уже удален", scenarioID)
	}

	return r.db.Model(&ds.BatteryScenario{}).Where("id = ?", scenarioID).Update("scenario_status", "deleted").Error
}

func (r *Repository) ToggleLikeAPI(userID uint, scenarioID uint, action int) (*ds.ScenarioLikeResponse, error) {
	var scenario ds.BatteryScenario
	err := r.db.Where("id = ? AND scenario_status != 'deleted'", scenarioID).First(&scenario).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("сценарий с id %d не найден или удален", scenarioID)
		}
		return nil, err
	}

	var existingLike ds.BatteryScenarioLike
	likeErr := r.db.Where("user_id = ? AND scenario_id = ?", userID, scenarioID).First(&existingLike).Error

	if action == 1 {
		if errors.Is(likeErr, gorm.ErrRecordNotFound) {
			newLike := ds.BatteryScenarioLike{
				UserID:     userID,
				ScenarioID: scenarioID,
				CreatedAt:  time.Now(),
			}
			if err := r.db.Create(&newLike).Error; err != nil {
				return nil, fmt.Errorf("не удалось поставить лайк: %w", err)
			}
		}
	} else if action == 0 {
		if likeErr == nil {
			if err := r.db.Where("user_id = ? AND scenario_id = ?", userID, scenarioID).Delete(&ds.BatteryScenarioLike{}).Error; err != nil {
				return nil, fmt.Errorf("не удалось отменить лайк: %w", err)
			}
		}
	} else {
		return nil, fmt.Errorf("недопустимое значение поля like (ожидается 0 или 1)")
	}

	likesCount := r.GetLikesCount(scenarioID)
	return &ds.ScenarioLikeResponse{
		ScenarioID: scenarioID,
		Liked:      action,
		LikesCount: likesCount,
	}, nil
}
