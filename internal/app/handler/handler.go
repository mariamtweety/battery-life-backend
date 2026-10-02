package handler

import (
	"net/http"

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
		ctx.Redirect(http.StatusFound, "/")
	})

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
