package minio

import (
	"bytes"
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// CreateOne создает один объект в бакете Minio.
// Метод принимает структуру fileData, которая содержит имя файла и его данные.
// В случае успешной загрузки данных в бакет, метод возвращает nil, иначе возвращает ошибку.
// Все операции выполняются в контексте задачи.
func (m *minioClient) CreateOne(file domain.FileDataType) (string, error) {
	// Генерация уникального идентификатора для нового объекта.
	objectID := uuid.New().String()

	// Создание потока данных для загрузки в бакет Minio.
	reader := bytes.NewReader(file.Data)

	// Загрузка данных в бакет Minio с использованием контекста для возможности отмены операции.
	_, err := m.mc.PutObject(context.Background(), m.minioConfig.BucketName, objectID, reader, int64(len(file.Data)), minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("ошибка при создании объекта %s: %v", file.FileName, err)
	}

	// Получение URL для загруженного объекта
	url, err := m.mc.PresignedGetObject(context.Background(), m.minioConfig.BucketName, objectID, time.Second*24*60*60, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка при создании URL для объекта %s: %v", file.FileName, err)
	}

	return url.String(), nil
}

// CreateMany создает несколько объектов в хранилище MinIO из переданных данных.
// Если происходит ошибка при создании объекта, метод возвращает ошибку,
// указывающую на неудачные объекты.
func (m *minioClient) CreateMany(data map[string]domain.FileDataType) ([]string, error) {
	urls := make([]string, 0, len(data)) // Массив для хранения URL-адресов

	ctx, cancel := context.WithCancel(context.Background()) // Создание контекста с возможностью отмены операции.
	defer cancel()                                          // Отложенный вызов функции отмены контекста при завершении функции CreateMany.

	// Создание канала для передачи URL-адресов с размером, равным количеству переданных данных.
	urlCh := make(chan string, len(data))

	var wg sync.WaitGroup // WaitGroup для ожидания завершения всех горутин.

	// Запуск горутин для создания каждого объекта.
	for objectID, file := range data {
		wg.Add(1) // Увеличение счетчика WaitGroup перед запуском каждой горутины.
		go func(objectID string, file domain.FileDataType) {
			defer wg.Done()                                                                                                                                // Уменьшение счетчика WaitGroup после завершения горутины.
			_, err := m.mc.PutObject(ctx, m.minioConfig.BucketName, objectID, bytes.NewReader(file.Data), int64(len(file.Data)), minio.PutObjectOptions{}) // Создание объекта в бакете MinIO.
			if err != nil {
				cancel() // Отмена операции при возникновении ошибки.
				return
			}

			// Получение URL для загруженного объекта
			url, err := m.mc.PresignedGetObject(ctx, m.minioConfig.BucketName, objectID, time.Second*24*60*60, nil)
			if err != nil {
				cancel() // Отмена операции при возникновении ошибки.
				return
			}

			urlCh <- url.String() // Отправка URL-адреса в канал с URL-адресами.
		}(objectID, file) // Передача данных объекта в анонимную горутину.
	}

	// Ожидание завершения всех горутин и закрытие канала с URL-адресами.
	go func() {
		wg.Wait()    // Блокировка до тех пор, пока счетчик WaitGroup не станет равным 0.
		close(urlCh) // Закрытие канала с URL-адресами после завершения всех горутин.
	}()

	// Сбор URL-адресов из канала.
	for url := range urlCh {
		urls = append(urls, url) // Добавление URL-адреса в массив URL-адресов.
	}

	return urls, nil
}
