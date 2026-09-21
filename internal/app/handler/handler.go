package handler

import (
	"battery-life/internal/app/repository"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

func (h *Handler) GetFeed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
		ctx.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Неверный ID"})
		return
	}

	next := ctx.Query("next")
	var scenario repository.Scenario

	if next == "true" {
		scenario, err = h.Repository.GetNextScenario(id)
	} else {
		scenario, err = h.Repository.GetScenarioByID(id)
	}

	if err != nil {
		logrus.Error(err)
		ctx.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Сценарий не найден"})
		return
	}

	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"scenario": scenario,
		"likes":    len(scenario.Likes),
	})
}

func (h *Handler) GetFeedDefault(ctx *gin.Context) {
	scenarios, err := h.Repository.GetPublishedScenarios()
	if err != nil || len(scenarios) == 0 {
		logrus.Error(err)
		ctx.HTML(http.StatusOK, "feed.html", gin.H{
			"scenario": nil,
			"likes":    0,
		})
		return
	}

	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"scenario": scenarios[0],
		"likes":    len(scenarios[0].Likes),
	})
}

func (h *Handler) GetDraft(ctx *gin.Context) {
	if idStr, selected := ctx.GetQuery("id"); selected {
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			ctx.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Неверный ID"})
			return
		}
		scenario, err := h.Repository.GetScenarioByID(id)
		if err != nil || scenario.Status == "deleted" {
			ctx.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Сценарий не найден"})
			return
		}
		scenario.Status = "draft"
		ctx.HTML(http.StatusOK, "add.html", gin.H{"scenario": scenario})
		return
	}

	scenario, err := h.Repository.GetDraftScenario()
	if err != nil {
		logrus.Error(err)
		ctx.HTML(http.StatusOK, "add.html", gin.H{
			"scenario": nil,
		})
		return
	}

	ctx.HTML(http.StatusOK, "add.html", gin.H{
		"scenario": scenario,
	})
}

func (h *Handler) GetScenariosList(ctx *gin.Context) {
	filterStr := strings.TrimSpace(ctx.Query("filter"))
	scenarios, err := h.Repository.GetPublishedScenarios()
	query := strings.ToLower(filterStr)
	maxHours := 17
	if raw, present := ctx.GetQuery("max_hours"); present {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 0 || parsed > 24 {
			ctx.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Время должно быть от 0 до 24 часов"})
			return
		}
		maxHours = parsed
	}

	if err != nil {
		logrus.Error(err)
	}

	type ScenarioWithLikes struct {
		repository.Scenario
		LikesCount int
	}

	var result []ScenarioWithLikes
	for _, s := range scenarios {
		if s.WorkTime > float64(maxHours) || !strings.Contains(strings.ToLower(s.Name), query) {
			continue
		}
		result = append(result, ScenarioWithLikes{
			Scenario:   s,
			LikesCount: len(s.Likes),
		})
	}

	ctx.HTML(http.StatusOK, "scenarios.html", gin.H{
		"scenarios": result,
		"filter":    filterStr,
		"maxHours":  maxHours,
	})
}
