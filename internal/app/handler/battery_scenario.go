package handler

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"battery-life/internal/app/ds"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const defaultUserID = 1

// GetFeed отображает сценарий в ленте по ID (или следующий при ?next=true)
func (h *Handler) GetFeed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		logrus.Errorf("GetFeed: неверный ID сценария %s", idStr)
		h.errorHandler(ctx, http.StatusBadRequest, "Неверный ID сценария", err)
		return
	}

	var scenario *ds.BatteryScenario
	if ctx.Query("next") == "true" {
		scenario, err = h.Repository.GetNextPublishedScenario(uint(id))
	} else {
		scenario, err = h.Repository.GetScenarioByID(uint(id))
	}

	if err != nil || scenario == nil {
		logrus.Warnf("GetFeed: сценарий с ID %d не найден или удален", id)
		h.errorHandler(ctx, http.StatusNotFound, "Сценарий удален или не найден", err)
		return
	}

	likesCount := h.Repository.GetLikesCount(scenario.ID)

	ctx.HTML(http.StatusOK, "feed.html", buildFeedData(scenario, likesCount))
}

// GetFeedDefault отображает первый опубликованный сценарий
func (h *Handler) GetFeedDefault(ctx *gin.Context) {
	scenario, err := h.Repository.GetFirstPublishedScenario()
	if err != nil || scenario == nil {
		logrus.Warn("GetFeedDefault: нет опубликованных сценариев")
		ctx.HTML(http.StatusOK, "feed.html", buildFeedData(nil, 0))
		return
	}

	likesCount := h.Repository.GetLikesCount(scenario.ID)

	ctx.HTML(http.StatusOK, "feed.html", buildFeedData(scenario, likesCount))
}

func buildFeedData(scenario *ds.BatteryScenario, likes int64) gin.H {
	data := gin.H{
		"scenario":     scenario,
		"likes":        likes,
		"is_long_desc": false,
		"short_desc":   "",
	}
	if scenario != nil {
		descRunes := []rune(scenario.ScenarioDescription)
		if len(descRunes) > 80 {
			data["is_long_desc"] = true
			data["short_desc"] = string(descRunes[:80])
		}
	}
	return data
}

// GetDraftScenario отображает страницу добавления / публикации черновика
func (h *Handler) GetDraftScenario(ctx *gin.Context) {
	// Поддержка перехода "Подробнее" из ленты по id
	if idStr, selected := ctx.GetQuery("id"); selected {
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			h.errorHandler(ctx, http.StatusBadRequest, "Неверный ID сценария", err)
			return
		}
		scenario, err := h.Repository.GetScenarioByID(uint(id))
		if err != nil || scenario == nil {
			h.errorHandler(ctx, http.StatusNotFound, "Сценарий удален или не найден", err)
			return
		}
		ctx.HTML(http.StatusOK, "add.html", gin.H{
			"scenario":  scenario,
			"has_draft": scenario.ScenarioStatus == "draft",
		})
		return
	}

	draft, err := h.Repository.GetDraftScenarioByUserID(defaultUserID)
	if err != nil {
		logrus.Errorf("GetDraftScenario: ошибка получения черновика: %v", err)
		h.errorHandler(ctx, http.StatusInternalServerError, "Ошибка при загрузке черновика", err)
		return
	}

	ctx.HTML(http.StatusOK, "add.html", gin.H{
		"scenario":  draft,
		"has_draft": draft != nil,
	})
}

// CreateDraftScenario создает новую карточку сценария (черновик) через ORM
func (h *Handler) CreateDraftScenario(ctx *gin.Context) {
	title := strings.TrimSpace(ctx.PostForm("scenario_title"))
	titleLen := utf8.RuneCountInString(title)
	if titleLen < 1 || titleLen > 24 {
		h.errorHandler(ctx, http.StatusBadRequest, "Длина названия сценария должна быть до 24 символов", nil)
		return
	}

	existingDraft, err := h.Repository.GetDraftScenarioByUserID(defaultUserID)
	if err != nil {
		logrus.Errorf("CreateDraftScenario: ошибка проверки существующего черновика: %v", err)
	}
	if existingDraft != nil {
		// У пользователя уже есть черновик - перенаправляем на страницу добавления
		ctx.Redirect(http.StatusFound, "/add")
		return
	}

	description := strings.TrimSpace(ctx.PostForm("scenario_description"))
	drainMah, _ := strconv.Atoi(ctx.PostForm("battery_drain_mah"))
	durationHours, _ := strconv.ParseFloat(ctx.PostForm("usage_duration_hours"), 64)

	scenario := &ds.BatteryScenario{
		ScenarioTitle:       title,
		ScenarioDescription: description,
		BatteryDrainMah:     drainMah,
		UsageDurationHours:  durationHours,
		CreatedByUserID:     defaultUserID,
		ImageURL:            strings.TrimSpace(ctx.PostForm("image_url")),
		VideoURL:            strings.TrimSpace(ctx.PostForm("video_url")),
	}

	err = h.Repository.CreateDraftScenario(scenario)
	if err != nil {
		logrus.Errorf("CreateDraftScenario: ошибка создания черновика: %v", err)
		h.errorHandler(ctx, http.StatusInternalServerError, "Не удалось создать черновик", err)
		return
	}

	ctx.Redirect(http.StatusFound, "/add")
}

// PublishScenario публикует карточку (смена статуса черновика на опубликован) через ORM
func (h *Handler) PublishScenario(ctx *gin.Context) {
	idStr := ctx.PostForm("scenario_id")
	if idStr == "" {
		idStr = ctx.Param("id")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		draft, draftErr := h.Repository.GetDraftScenarioByUserID(defaultUserID)
		if draftErr != nil || draft == nil {
			h.errorHandler(ctx, http.StatusBadRequest, "Неверный ID сценария для публикации", err)
			return
		}
		id = int(draft.ID)
	}

	description := strings.TrimSpace(ctx.PostForm("scenario_description"))
	descLen := utf8.RuneCountInString(description)
	if descLen < 1 || descLen > 1000 {
		h.errorHandler(ctx, http.StatusBadRequest, "Длина описания должна быть от 1 до 1000 символов", nil)
		return
	}

	drainMah, drainErr := strconv.Atoi(ctx.PostForm("battery_drain_mah"))
	durationHours, durErr := strconv.ParseFloat(ctx.PostForm("usage_duration_hours"), 64)

	if drainErr != nil || drainMah < 1 || drainMah > 9999 || durErr != nil || durationHours <= 0 || durationHours > 24 {
		h.errorHandler(ctx, http.StatusBadRequest, "Расход заряда должен быть от 1 до 9999 мА*ч, а время от 0.1 до 24 ч", nil)
		return
	}

	err = h.Repository.PublishScenario(uint(id), description, drainMah, durationHours)
	if err != nil {
		logrus.Errorf("PublishScenario: ошибка публикации сценария %d: %v", id, err)
		h.errorHandler(ctx, http.StatusInternalServerError, "Не удалось опубликовать сценарий", err)
		return
	}

	ctx.Redirect(http.StatusFound, "/scenarios")
}

// GetScenariosList отображает плитку опубликованных карточек с поиском и фильтром
func (h *Handler) GetScenariosList(ctx *gin.Context) {
	filterStr := strings.TrimSpace(ctx.Query("filter"))
	maxHours := 24.0

	if raw, present := ctx.GetQuery("max_hours"); present && raw != "" {
		if parsed, parseErr := strconv.ParseFloat(raw, 64); parseErr == nil && parsed >= 0 && parsed <= 24 {
			maxHours = parsed
		}
	}

	scenarios, err := h.Repository.GetScenariosWithLikes(filterStr, maxHours)
	if err != nil {
		logrus.Errorf("GetScenariosList: ошибка выборки сценариев: %v", err)
		h.errorHandler(ctx, http.StatusInternalServerError, "Ошибка при загрузке сценариев", err)
		return
	}

	ctx.HTML(http.StatusOK, "scenarios.html", gin.H{
		"scenarios": scenarios,
		"filter":    filterStr,
		"maxHours":  int(maxHours),
	})
}

// DeleteScenario выполняет логическое удаление карточки через SQL UPDATE без ORM
func (h *Handler) DeleteScenario(ctx *gin.Context) {
	idStr := ctx.PostForm("scenario_id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		logrus.Errorf("DeleteScenario: неверный ID %s", idStr)
		h.errorHandler(ctx, http.StatusBadRequest, "Неверный ID для удаления", err)
		return
	}

	// Вызов логического удаления через SQL запрос UPDATE (без ORM)
	err = h.Repository.DeleteScenarioSQL(uint(id))
	if err != nil {
		logrus.Errorf("DeleteScenario: ошибка SQL UPDATE: %v", err)
		h.errorHandler(ctx, http.StatusInternalServerError, "Не удалось удалить сценарий", err)
		return
	}

	ctx.Redirect(http.StatusFound, "/scenarios")
}
