package api

import (
	"battery-life/internal/app/handler"
	"battery-life/internal/app/repository"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func StartServer() {
	log.Println("Server start up")

	repo, err := repository.NewRepository()
	if err != nil {
		logrus.Error("ошибка инициализации репозитория")
	}

	h := handler.NewHandler(repo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/", h.GetFeedDefault)
	r.GET("/feed/:id", h.GetFeed)
	r.GET("/add", h.GetDraft)
	r.GET("/scenarios", h.GetScenariosList)

	r.Run()

	log.Println("Server down")
}
