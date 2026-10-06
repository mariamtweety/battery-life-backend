package handler

import (
	"net/http"
	"strings"

	"battery-life/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		if strings.HasPrefix(ctx.Request.URL.Path, "/api") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "маршрут API не найден"})
			return
		}
		ctx.Redirect(http.StatusFound, "/")
	})

	// SSR routes from Lab 1 & 2
	router.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/scenarios/feed")
	})
	router.GET("/scenarios/feed", h.GetDefaultScenarioFeed)
	router.GET("/scenarios/feed/:id", h.GetScenarioFeed)
	router.GET("/scenarios/draft", h.GetDraftScenario)
	router.GET("/scenarios", h.GetScenarioCatalog)

	router.POST("/scenarios/draft", h.CreateDraftScenario)
	router.POST("/scenarios/draft/publish", h.PublishScenario)
	router.POST("/scenarios/delete", h.DeleteScenario)

	// --- REST API routes for Lab 3 ---
	api := router.Group("/api")
	{
		// Scenario domain
		api.GET("/scenarios", h.GetScenariosAPI)
		api.GET("/scenarios/feed", h.GetScenarioFeedAPI)
		api.GET("/scenarios/feed/:id", h.GetScenarioFeedAPI)
		api.GET("/scenarios/draft", h.GetDraftScenarioAPI)
		api.POST("/scenarios", h.CreateDraftScenarioAPI)
		api.PUT("/scenarios/:id/publish", h.PublishScenarioAPI)
		api.PUT("/scenarios/:id", h.PublishScenarioAPI)
		api.DELETE("/scenarios/:id", h.DeleteScenarioAPI)
		api.POST("/scenarios/:id/like", h.ToggleLikeAPI)

		// User domain
		api.POST("/users/register", h.RegisterUserAPI)
		api.POST("/users/login", h.LoginUserAPI)
		api.POST("/users/logout", h.LogoutUserAPI)
		api.GET("/users/:id", h.GetUserScenariosAPI)
	}
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources")
}

func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, message string, err error) {
	if err != nil {
		logrus.Errorf("%s: %v", message, err)
	} else {
		logrus.Error(message)
	}
	if errorStatusCode == http.StatusNotFound {
		ctx.Redirect(http.StatusFound, "/")
		return
	}
	ctx.String(errorStatusCode, message)
}
