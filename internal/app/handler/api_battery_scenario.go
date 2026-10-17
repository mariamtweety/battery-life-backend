package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"battery-life/internal/app/ds"
	"battery-life/internal/app/singleton"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func apiError(ctx *gin.Context, statusCode int, message string) {
	logrus.Warnf("REST API Error [%d]: %s", statusCode, message)
	ctx.JSON(statusCode, gin.H{
		"error": message,
	})
}

func (h *Handler) GetScenariosAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()

	filterStr := strings.TrimSpace(ctx.Query("search"))
	if filterStr == "" {
		filterStr = strings.TrimSpace(ctx.Query("filter"))
	}
	if filterStr == "" {
		filterStr = strings.TrimSpace(ctx.Query("title"))
	}

	maxHours := 0.0
	if raw := strings.TrimSpace(ctx.Query("max_hours")); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed > 0 {
			maxHours = parsed
		}
	}

	scenarios, err := h.Repository.GetPublishedScenariosAPI(currentUserID, filterStr, maxHours)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, "Ошибка получения списка сценариев: "+err.Error())
		return
	}

	ctx.JSON(http.StatusOK, scenarios)
}

func (h *Handler) GetScenarioFeedAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()
	idStr := ctx.Query("id")
	if idStr == "" {
		apiError(ctx, http.StatusBadRequest, "Параметр id обязателен")
		return
	}

	idVal, err := strconv.Atoi(idStr)
	if err != nil || idVal <= 0 {
		apiError(ctx, http.StatusBadRequest, "Неверный ID сценария")
		return
	}
	uVal := uint(idVal)

	isNext := ctx.Query("next") == "true"

	scenario, err := h.Repository.GetFeedScenarioAPI(currentUserID, &uVal, isNext)
	if err != nil {
		apiError(ctx, http.StatusNotFound, "Сценарий в ленте не найден")
		return
	}

	ctx.JSON(http.StatusOK, scenario)
}

func (h *Handler) GetDraftScenarioAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()

	draft, err := h.Repository.GetDraftScenarioAPI(currentUserID)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, "Ошибка получения черновика: "+err.Error())
		return
	}
	if draft == nil {
		apiError(ctx, http.StatusNotFound, "У текущего пользователя нет черновика")
		return
	}

	ctx.JSON(http.StatusOK, draft)
}

func (h *Handler) CreateDraftScenarioAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()

	if err := ctx.Request.ParseMultipartForm(50 << 20); err != nil && err != http.ErrNotMultipart {
		apiError(ctx, http.StatusBadRequest, "Ошибка парсинга формы: "+err.Error())
		return
	}

	title := strings.TrimSpace(ctx.PostForm("scenario_title"))
	if title == "" {
		title = strings.TrimSpace(ctx.PostForm("title"))
	}
	titleLen := utf8.RuneCountInString(title)
	if titleLen < 1 || titleLen > 24 {
		apiError(ctx, http.StatusBadRequest, "Название сценария обязательно и должно содержать от 1 до 24 символов")
		return
	}

	description := strings.TrimSpace(ctx.PostForm("scenario_description"))
	if description == "" {
		description = strings.TrimSpace(ctx.PostForm("description"))
	}
	if utf8.RuneCountInString(description) > 1000 {
		apiError(ctx, http.StatusBadRequest, "Описание сценария не должно превышать 1000 символов")
		return
	}

	drainMah, _ := strconv.Atoi(ctx.PostForm("battery_drain_mah"))
	durationHours, _ := strconv.ParseFloat(ctx.PostForm("usage_duration_hours"), 64)

	scenario := &ds.BatteryScenario{
		ScenarioTitle:       title,
		ScenarioDescription: description,
		BatteryDrainMah:     drainMah,
		UsageDurationHours:  durationHours,
		CreatedByUserID:     currentUserID,
	}

	savedScenario, err := h.Repository.CreateOrUpdateDraftScenario(scenario)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, "Не удалось сохранить черновик сценария: "+err.Error())
		return
	}

	imgHeader, imgErr := ctx.FormFile("image")
	if imgErr != nil {
		imgHeader, imgErr = ctx.FormFile("scenario_image")
	}
	if imgErr != nil {
		imgHeader, imgErr = ctx.FormFile("pic")
	}

	var imageURL string
	if imgErr == nil && imgHeader != nil {
		uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		url, err := h.Repository.UploadScenarioMedia(uploadCtx, imgHeader, "image", savedScenario.ID)
		if err != nil {
			logrus.Errorf("UploadScenarioMedia (image): %v", err)
		} else {
			imageURL = url
		}
	}

	vidHeader, vidErr := ctx.FormFile("video")
	if vidErr != nil {
		vidHeader, vidErr = ctx.FormFile("scenario_video")
	}
	if vidErr != nil {
		vidHeader, vidErr = ctx.FormFile("clip")
	}

	var videoURL string
	if vidErr == nil && vidHeader != nil {
		uploadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		url, err := h.Repository.UploadScenarioMedia(uploadCtx, vidHeader, "video", savedScenario.ID)
		if err != nil {
			logrus.Errorf("UploadScenarioMedia (video): %v", err)
		} else {
			videoURL = url
		}
	}

	if imageURL != "" || videoURL != "" {
		if err := h.Repository.UpdateScenarioMedia(savedScenario.ID, imageURL, videoURL); err != nil {
			logrus.Errorf("UpdateScenarioMedia: %v", err)
		}
		if imageURL != "" {
			savedScenario.ImageURL = imageURL
		}
		if videoURL != "" {
			savedScenario.VideoURL = videoURL
		}
	}

	fullDraft, err := h.Repository.GetDraftScenarioAPI(currentUserID)
	if err == nil && fullDraft != nil {
		ctx.JSON(http.StatusCreated, fullDraft)
		return
	}

	ctx.JSON(http.StatusCreated, savedScenario)
}

func (h *Handler) PublishScenarioAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()

	idStr := ctx.Param("id")
	idVal, err := strconv.Atoi(idStr)
	if err != nil || idVal <= 0 {
		apiError(ctx, http.StatusBadRequest, "Неверный ID сценария для публикации")
		return
	}

	var req ds.ScenarioPublishRequest

	if ctx.ContentType() == "application/json" {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apiError(ctx, http.StatusBadRequest, "Неверный формат JSON: "+err.Error())
			return
		}
	} else {
		req.ScenarioTitle = strings.TrimSpace(ctx.PostForm("scenario_title"))
		req.ScenarioDescription = strings.TrimSpace(ctx.PostForm("scenario_description"))
		req.BatteryDrainMah, _ = strconv.Atoi(ctx.PostForm("battery_drain_mah"))
		req.UsageDurationHours, _ = strconv.ParseFloat(ctx.PostForm("usage_duration_hours"), 64)
	}

	descLen := utf8.RuneCountInString(req.ScenarioDescription)
	if descLen < 1 || descLen > 1000 {
		apiError(ctx, http.StatusBadRequest, "Описание сценария обязательно и должно быть от 1 до 1000 символов")
		return
	}

	if req.BatteryDrainMah < 1 || req.BatteryDrainMah > 9999 {
		apiError(ctx, http.StatusBadRequest, "Расход заряда должен быть в диапазоне от 1 до 9999 мА*ч")
		return
	}

	if req.UsageDurationHours <= 0 || req.UsageDurationHours > 24 {
		apiError(ctx, http.StatusBadRequest, "Время использования должно быть в диапазоне от 0.1 до 24 часов")
		return
	}

	publishedScenario, err := h.Repository.PublishScenarioAPI(uint(idVal), currentUserID, req)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, "Ошибка публикации сценария: "+err.Error())
		return
	}

	ctx.JSON(http.StatusOK, publishedScenario)
}

func (h *Handler) DeleteScenarioAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()

	idStr := ctx.Param("id")
	idVal, err := strconv.Atoi(idStr)
	if err != nil || idVal <= 0 {
		apiError(ctx, http.StatusBadRequest, "Неверный ID сценария для удаления")
		return
	}

	if err := h.Repository.SoftDeleteScenarioAPI(uint(idVal), currentUserID); err != nil {
		apiError(ctx, http.StatusBadRequest, "Не удалось удалить сценарий: "+err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "сценарий успешно удален",
		"id":      uint(idVal),
	})
}

func (h *Handler) ToggleLikeAPI(ctx *gin.Context) {
	currentUserID := singleton.GetCurrentUserID()

	idStr := ctx.Param("id")
	idVal, err := strconv.Atoi(idStr)
	if err != nil || idVal <= 0 {
		apiError(ctx, http.StatusBadRequest, "Неверный ID сценария")
		return
	}

	action := 1
	if ctx.ContentType() == "application/json" {
		var req ds.ScenarioLikeRequest
		if err := ctx.ShouldBindJSON(&req); err == nil {
			action = req.Like
		}
	}

	if action != 0 && action != 1 {
		apiError(ctx, http.StatusBadRequest, "Поле like должно принимать значение 1 (поставить) или 0 (отменить)")
		return
	}

	res, err := h.Repository.ToggleLikeAPI(currentUserID, uint(idVal), action)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, "Ошибка обновления лайка: "+err.Error())
		return
	}

	ctx.JSON(http.StatusOK, res)
}
