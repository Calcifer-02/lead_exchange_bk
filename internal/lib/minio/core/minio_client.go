package minio

import (
	"context"
	"lead_exchange/internal/config"
	"lead_exchange/internal/domain"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client интерфейс для взаимодействия с Minio
type Client interface {
	InitMinio(MinioConfig config.MinioConfig) error              // Метод для инициализации подключения к Minio
	CreateOne(file domain.FileDataType) (string, error)          // Метод для создания одного объекта в бакете Minio
	CreateMany(map[string]domain.FileDataType) ([]string, error) // Метод для создания нескольких объектов в бакете Minio
}

// minioClient реализация интерфейса MinioClient
type minioClient struct {
	mc          *minio.Client
	minioConfig config.MinioConfig
}

// NewMinioClient создает новый экземпляр Minio Client
func NewMinioClient() Client {
	return &minioClient{} // Возвращает новый экземпляр minioClient с указанным именем бакета
}

// InitMinio подключается к Minio и создает бакет, если не существует
// Бакет - это контейнер для хранения объектов в Minio. Он представляет собой пространство имен, в котором можно хранить и организовывать файлы и папки.
func (m *minioClient) InitMinio(MinioConfig config.MinioConfig) error {
	// Создание контекста с возможностью отмены операции
	ctx := context.Background()

	m.minioConfig = MinioConfig

	// Подключение к Minio с использованием имени пользователя и пароля
	client, err := minio.New(MinioConfig.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(MinioConfig.MinioRootUser, MinioConfig.MinioRootPassword, ""),
		Secure: MinioConfig.MinioUseSSL,
	})
	if err != nil {
		return err
	}

	// Установка подключения Minio
	m.mc = client

	// Проверка наличия бакета и его создание, если не существует
	exists, err := m.mc.BucketExists(ctx, MinioConfig.BucketName)
	if err != nil {
		return err
	}
	if !exists {
		err := m.mc.MakeBucket(ctx, MinioConfig.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
