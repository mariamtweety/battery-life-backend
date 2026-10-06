package main

import (
	"os"

	"battery-life/internal/app/config"
	"battery-life/internal/app/dsn"
	"battery-life/internal/app/handler"
	"battery-life/internal/app/repository"
	"battery-life/internal/pkg"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	router := gin.Default()
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

	hand := handler.NewHandler(rep)

	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
