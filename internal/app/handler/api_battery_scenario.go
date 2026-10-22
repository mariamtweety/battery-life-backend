package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"battery-life/internal/app/ds"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func apiError(ctx *gin.Context, statusCode int, message string) {
	logrus.Warnf("REST API Error [%d]: %s", statusCode, message)
	ctx.JSON(statusCode, gin.H{
		"error": message,
	})
}

// GetScenariosAPI godoc
//
//	@Summary		Список опубликованных услуг
//	@Description	Доступно без авторизации. Если передан JWT, заполняются is_liked и is_creator.
//	@Tags			scenarios
//	@Produce		json
//	@Param			search		query		string	false	"Фильтр по названию"
//	@Param			max_hours	query		number	false	"Максимальная длительность использования, ч"
//	@Success		200			{array}		ds.ScenarioListItemSerializer
//	@Failure		500			{object}	ds.ErrorResponse
//	@Router			/scenarios [get]
func (h *Handler) GetScenariosAPI(ctx *gin.Context) {
	userID := currentUserID(ctx)

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

	scenarios, err := h.Repository.GetPublishedScenariosAPI(userID, filterStr, maxHours)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, "Ошибка получения списка сценариев: "+err.Error())
		return
	}

	ctx.JSON(http.StatusOK, scenarios)
}

// GetScenarioFeedAPI godoc
//
//	@Summary		Лента услуг
//	@Description	Без id — первая опубликованная услуга. С id — услуга по id. С id и next=true — следующая, с заворотом в начало ленты. Доступно без авторизации; is_liked/is_creator заполняются при наличии JWT.
//	@Tags			scenarios
//	@Produce		json
//	@Param			id		path		int		false	"ID текущей услуги"
//	@Param			next	query		bool	false	"Получить следующую услугу"
//	@Success		200		{object}	ds.ScenarioFeedItemSerializer
//	@Failure		400		{object}	ds.ErrorResponse
//	@Failure		404		{object}	ds.ErrorResponse
//	@Router			/scenarios/feed/{id} [get]
func (h *Handler) GetScenarioFeedAPI(ctx *gin.Context) {
	userID := currentUserID(ctx)

	idStr := ctx.Param("id")
	if idStr == "" {
		idStr = ctx.Query("id")
	}

	var uVal *uint
	if idStr != "" {
		idVal, err := strconv.Atoi(idStr)
		if err != nil || idVal <= 0 {
			apiError(ctx, http.StatusBadRequest, "Неверный ID сценария")
			return
		}
		val := uint(idVal)
		uVal = &val
	}

	isNext := ctx.Query("next") == "true"

	scenario, err := h.Repository.GetFeedScenarioAPI(userID, uVal, isNext)
	if err != nil {
		apiError(ctx, http.StatusNotFound, "Сценарий в ленте не найден")
		return
	}

	ctx.JSON(http.StatusOK, scenario)
}

// GetDraftScenarioAPI godoc
//
//	@Summary		Черновик текущего пользователя
//	@Description	Возвращает единственный черновик авторизованного пользователя. ID с клиента не принимается.
//	@Tags			scenarios
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	ds.ScenarioDraftDetailSerializer
//	@Failure		401	{object}	ds.ErrorResponse
//	@Failure		404	{object}	ds.ErrorResponse
//	@Router			/scenarios/draft [get]
func (h *Handler) GetDraftScenarioAPI(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftScenarioAPI(currentUserID(ctx))
	if err != nil {
		writeRepoError(ctx, err, "Ошибка получения черновика")
		return
	}
	if draft == nil {
		apiError(ctx, http.StatusNotFound, "У текущего пользователя нет черновика")
		return
	}

	ctx.JSON(http.StatusOK, draft)
}

// CreateDraftScenarioAPI godoc
//
//	@Summary		Создать (или обновить) черновик услуги
//	@Description	Принимает multipart/form-data. Файлы image и video сохраняются в MinIO, создатель берётся из JWT.
//	@Tags			scenarios
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			scenario_title	formData	string	false	"Название услуги (1-24 символа)"
//	@Param			image			formData	file	false	"Изображение"
//	@Param			video			formData	file	false	"Видео"
//	@Success		201				{object}	ds.ScenarioDraftResponse
//	@Failure		400				{object}	ds.ErrorResponse
//	@Failure		401				{object}	ds.ErrorResponse
//	@Failure		500				{object}	ds.ErrorResponse
//	@Router			/scenarios [post]
func (h *Handler) CreateDraftScenarioAPI(ctx *gin.Context) {
	userID := currentUserID(ctx)
	if userID == 0 {
		apiError(ctx, http.StatusUnauthorized, "Требуется авторизация")
		return
	}

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
		// Создатель всегда берётся из JWT, а не из тела запроса.
		CreatedByUserID: userID,
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

	resp := ds.ScenarioDraftResponse{
		ID:            savedScenario.ID,
		ScenarioTitle: savedScenario.ScenarioTitle,
		ImageURL:      savedScenario.ImageURL,
		VideoURL:      savedScenario.VideoURL,
	}

	ctx.JSON(http.StatusCreated, resp)
}

// PublishScenarioAPI godoc
//
//	@Summary		Опубликовать свой черновик
//	@Description	Меняет статус draft → published. Доступно только создателю. Для гостя — 401.
//	@Tags			scenarios
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID черновика"
//	@Param			data	body		ds.ScenarioPublishRequest	true	"Описание и поля по теме"
//	@Success		200		{object}	ds.ScenarioDraftDetailSerializer
//	@Failure		400		{object}	ds.ErrorResponse
//	@Failure		401		{object}	ds.ErrorResponse
//	@Failure		403		{object}	ds.ErrorResponse
//	@Failure		404		{object}	ds.ErrorResponse
//	@Failure		409		{object}	ds.ErrorResponse
//	@Router			/scenarios/{id}/publish [put]
func (h *Handler) PublishScenarioAPI(ctx *gin.Context) {
	userID := currentUserID(ctx)

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

	publishedScenario, err := h.Repository.PublishScenarioAPI(uint(idVal), userID, req)
	if err != nil {
		writeRepoError(ctx, err, "Ошибка публикации сценария")
		return
	}

	ctx.JSON(http.StatusOK, publishedScenario)
}

// DeleteScenarioAPI godoc
//
//	@Summary		Логически удалить услугу
//	@Description	Меняет статус на deleted (SQL UPDATE, без каскадного удаления). Создатель удаляет свои услуги, модератор — любые.
//	@Tags			scenarios
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID услуги"
//	@Success		200	{object}	ds.ScenarioDeleteResponse
//	@Failure		400	{object}	ds.ErrorResponse
//	@Failure		401	{object}	ds.ErrorResponse
//	@Failure		403	{object}	ds.ErrorResponse
//	@Failure		404	{object}	ds.ErrorResponse
//	@Failure		409	{object}	ds.ErrorResponse
//	@Router			/scenarios/{id} [delete]
func (h *Handler) DeleteScenarioAPI(ctx *gin.Context) {
	userID := currentUserID(ctx)

	idStr := ctx.Param("id")
	idVal, err := strconv.Atoi(idStr)
	if err != nil || idVal <= 0 {
		apiError(ctx, http.StatusBadRequest, "Неверный ID сценария для удаления")
		return
	}

	if err := h.Repository.SoftDeleteScenarioAPI(uint(idVal), userID, isModerator(ctx)); err != nil {
		writeRepoError(ctx, err, "Не удалось удалить сценарий")
		return
	}

	ctx.JSON(http.StatusOK, ds.ScenarioDeleteResponse{
		ID:      uint(idVal),
		Message: "сценарий успешно удален",
	})
}

// ToggleLikeAPI godoc
//
//	@Summary		Поставить или отменить лайк
//	@Description	like=1 ставит лайк текущего пользователя, like=0 отменяет.
//	@Tags			scenarios
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"ID услуги"
//	@Param			like	body		ds.ScenarioLikeRequest	true	"1 — поставить, 0 — отменить"
//	@Success		200		{object}	ds.ScenarioLikeResponse
//	@Failure		400		{object}	ds.ErrorResponse
//	@Failure		401		{object}	ds.ErrorResponse
//	@Failure		404		{object}	ds.ErrorResponse
//	@Router			/scenarios/{id}/like [post]
func (h *Handler) ToggleLikeAPI(ctx *gin.Context) {
	userID := currentUserID(ctx)

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

	res, err := h.Repository.ToggleLikeAPI(userID, uint(idVal), action)
	if err != nil {
		writeRepoError(ctx, err, "Ошибка обновления лайка")
		return
	}

	ctx.JSON(http.StatusOK, res)
}
