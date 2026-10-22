package main

import (
	"context"
	"os"

	_ "battery-life/docs"
	"battery-life/internal/app/auth"
	"battery-life/internal/app/config"
	"battery-life/internal/app/dsn"
	"battery-life/internal/app/handler"
	"battery-life/internal/app/redisclient"
	"battery-life/internal/app/repository"
	"battery-life/internal/pkg"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @title			Battery Life API
// @version		4.0
// @description	Бэкенд SPA «Battery Life». ЛР4: JWT-аутентификация, ролевая модель, blacklist токенов в Redis, Swagger.
// @description	Публичное чтение доступно без токена. Методы создателя требуют заголовок Authorization: Bearer <JWT>.
// @contact.name	Battery Life API Support
// @contact.email	support@battery-life.local
// @license.name	AS IS (NO WARRANTY)
// @host			localhost:8080
// @schemes		http https
// @BasePath		/api
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description					Введите "Bearer " и JWT, полученный в POST /api/users/login.
func main() {
	ctx := context.Background()

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.FromEnv()
	logrus.Infof("Connecting with Postgres DSN: %s", postgresString)

	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	rep, errRep := repository.New(&repository.RepositorySettings{
		PostgresDSN:     postgresString,
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucketName: os.Getenv("MINIO_BUCKET_NAME"),
		MinioUseSSL:     useSSL,
		MinioPublicURL:  os.Getenv("MINIO_PUBLIC_URL"),
	})
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	redisClient, err := redisclient.New(ctx, conf.Redis)
	if err != nil {
		logrus.Fatalf("error initializing Redis: %v", err)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logrus.Warnf("error closing Redis: %v", closeErr)
		}
	}()

	authManager := auth.New(conf.JWT, redisClient)
	hand := handler.NewHandler(rep, authManager, conf)

	router := gin.Default()
	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
