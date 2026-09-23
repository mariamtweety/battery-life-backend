package handler

import (
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
	// 3 GET метода
	router.GET("/", h.GetFeedDefault)
	router.GET("/feed/:id", h.GetFeed)
	router.GET("/add", h.GetDraftScenario)
	router.GET("/scenarios", h.GetScenariosList)

	// POST методы через ORM
	router.POST("/add", h.CreateDraftScenario)
	router.POST("/scenarios/create", h.CreateDraftScenario)
	router.POST("/add/publish", h.PublishScenario)
	router.POST("/scenarios/:id/publish", h.PublishScenario)

	// POST метод логического удаления через SQL UPDATE (без ORM)
	router.POST("/scenarios/delete", h.DeleteScenario)
	router.POST("/delete-scenario", h.DeleteScenario)
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
	ctx.HTML(errorStatusCode, "error.html", gin.H{
		"error": message,
	})
}
