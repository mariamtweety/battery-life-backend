package handler

import (
	"net/http"

	"battery-life/internal/app/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "маршрут API не найден"})
	})

	api := router.Group("/api")
	{
		api.GET("/scenarios", h.GetScenariosAPI)
		api.GET("/scenarios/feed", h.GetScenarioFeedAPI)
		api.GET("/scenarios/feed/:id", h.GetScenarioFeedAPI)
		api.GET("/scenarios/draft", h.GetDraftScenarioAPI)
		api.POST("/scenarios", h.CreateDraftScenarioAPI)
		api.PUT("/scenarios/:id/publish", h.PublishScenarioAPI)
		api.DELETE("/scenarios/:id", h.DeleteScenarioAPI)
		api.POST("/scenarios/:id/like", h.ToggleLikeAPI)

		api.POST("/users/register", h.RegisterUserAPI)
		api.POST("/users/login", h.LoginUserAPI)
		api.POST("/users/logout", h.LogoutUserAPI)
	}
}
