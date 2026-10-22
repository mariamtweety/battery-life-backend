package handler

import (
	"net/http"

	"battery-life/internal/app/auth"
	"battery-life/internal/app/config"
	"battery-life/internal/app/repository"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handler struct {
	Repository *repository.Repository
	Auth       *auth.Manager
	Config     *config.Config
}

func NewHandler(r *repository.Repository, a *auth.Manager, cfg *config.Config) *Handler {
	return &Handler{
		Repository: r,
		Auth:       a,
		Config:     cfg,
	}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.Use(corsMiddleware())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "маршрут API не найден"})
	})

	api := router.Group("/api")
	{
		// Публичное чтение: доступно без токена (гостю), но если токен передан —
		// в ответ добавляются is_liked/is_creator для текущего пользователя.
		public := api.Group("")
		public.Use(h.WithOptionalAuth())
		{
			public.GET("/scenarios", h.GetScenariosAPI)
			public.GET("/scenarios/feed", h.GetScenarioFeedAPI)
			public.GET("/scenarios/feed/:id", h.GetScenarioFeedAPI)
		}

		// Аутентификация.
		api.POST("/users/register", h.RegisterUserAPI)
		api.POST("/users/login", h.LoginUserAPI)

		// Методы создателя: только с валидным JWT, не попавшим в blacklist.
		protected := api.Group("")
		protected.Use(h.WithAuthCheck())
		{
			protected.GET("/users/me", h.CurrentUserAPI)
			protected.POST("/users/logout", h.LogoutUserAPI)

			protected.GET("/scenarios/draft", h.GetDraftScenarioAPI)
			protected.POST("/scenarios", h.CreateDraftScenarioAPI)
			protected.PUT("/scenarios/:id/publish", h.PublishScenarioAPI)
			protected.DELETE("/scenarios/:id", h.DeleteScenarioAPI)
			protected.POST("/scenarios/:id/like", h.ToggleLikeAPI)
		}
	}
}

// corsMiddleware разрешает запросы SPA с другим origin (в том числе с cookie).
func corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if origin := ctx.GetHeader("Origin"); origin != "" {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
			ctx.Header("Vary", "Origin")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
