package repository

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"battery-life/internal/app/ds"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type RepositorySettings struct {
	PostgresDSN     string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucketName string
	MinioUseSSL     bool
	MinioPublicURL  string
}

type Repository struct {
	db              *gorm.DB
	minio           *minio.Client
	minioBucketName string
	minioPublicURL  string
}

type ScenarioWithLikes struct {
	ds.BatteryScenario
	LikesCount int64
}

func New(settings *RepositorySettings) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(settings.PostgresDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к Postgres: %w", err)
	}

	var minioClient *minio.Client
	if settings.MinioEndpoint != "" {
		minioClient, err = minio.New(settings.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(settings.MinioAccessKey, settings.MinioSecretKey, ""),
			Secure: settings.MinioUseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("ошибка инициализации MinIO клиента: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bucketName := settings.MinioBucketName
		if bucketName == "" {
			bucketName = "scenarios"
		}

		exists, bucketErr := minioClient.BucketExists(ctx, bucketName)
		if bucketErr != nil {
			logrus.Warnf("Проверка бакета MinIO '%s': %v", bucketName, bucketErr)
		} else if !exists {
			if makeErr := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); makeErr != nil {
				logrus.Warnf("Не удалось создать бакет MinIO '%s': %v", bucketName, makeErr)
			} else {
				logrus.Infof("Бакет MinIO '%s' успешно создан", bucketName)
			}
		}

		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, bucketName)
		_ = minioClient.SetBucketPolicy(ctx, bucketName, policy)
	}

	publicURL := strings.TrimRight(settings.MinioPublicURL, "/")
	if publicURL == "" {
		schema := "http"
		if settings.MinioUseSSL {
			schema = "https"
		}
		publicURL = fmt.Sprintf("%s://%s", schema, settings.MinioEndpoint)
	}

	return &Repository{
		db:              db,
		minio:           minioClient,
		minioBucketName: settings.MinioBucketName,
		minioPublicURL:  publicURL,
	}, nil
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) MinIO() *minio.Client {
	return r.minio
}

func (r *Repository) BucketName() string {
	return r.minioBucketName
}

func (r *Repository) PublicURL() string {
	return r.minioPublicURL
}

// GenerateLatinFilename creates a safe Latin filename for uploaded scenario media
func GenerateLatinFilename(mediaType string, scenarioID uint, originalName string) string {
	ext := filepath.Ext(originalName)
	if ext == "" {
		if mediaType == "image" {
			ext = ".png"
		} else if mediaType == "video" {
			ext = ".mp4"
		} else {
			ext = ".bin"
		}
	}
	cleanExt := ""
	for _, ch := range strings.ToLower(ext) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' {
			cleanExt += string(ch)
		}
	}
	if cleanExt == "" {
		cleanExt = ".dat"
	}
	return fmt.Sprintf("scenario_%d_%s_%d%s", scenarioID, mediaType, time.Now().UnixNano(), cleanExt)
}

// UploadScenarioMedia uploads an image or video file to MinIO with Latin filename and returns its URL
func (r *Repository) UploadScenarioMedia(ctx context.Context, header *multipart.FileHeader, mediaType string, scenarioID uint) (string, error) {
	if r.minio == nil {
		return "", fmt.Errorf("клиент MinIO не настроен")
	}

	filename := GenerateLatinFilename(mediaType, scenarioID, header.Filename)

	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, readErr := file.Read(buf)
	if readErr != nil && readErr != io.EOF {
		return "", fmt.Errorf("ошибка чтения содержимого файла: %w", readErr)
	}

	detectedType := http.DetectContentType(buf[:n])
	contentType := detectedType
	if detectedType == "application/octet-stream" && header.Header.Get("Content-Type") != "" {
		contentType = header.Header.Get("Content-Type")
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("ошибка перемотки файла: %w", err)
	}

	_, err = r.minio.PutObject(ctx, r.minioBucketName, filename, file, header.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения файла в MinIO: %w", err)
	}

	fileURL := fmt.Sprintf("%s/%s/%s", r.minioPublicURL, r.minioBucketName, filename)
	logrus.Infof("Uploaded media to MinIO: %s (Content-Type: %s)", fileURL, contentType)
	return fileURL, nil
}
