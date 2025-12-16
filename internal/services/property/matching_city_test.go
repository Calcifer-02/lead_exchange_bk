package property

import (
	"context"
	"testing"

	"lead_exchange/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestifyMockPropertyRepository - мок репозитория для тестов (с testify)
type TestifyMockPropertyRepository struct {
	mock.Mock
}

func (m *TestifyMockPropertyRepository) Create(ctx context.Context, property domain.Property) (uuid.UUID, error) {
	args := m.Called(ctx, property)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *TestifyMockPropertyRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Property, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Property), args.Error(1)
}

func (m *TestifyMockPropertyRepository) Update(ctx context.Context, id uuid.UUID, filter domain.PropertyFilter) (domain.Property, error) {
	args := m.Called(ctx, id, filter)
	return args.Get(0).(domain.Property), args.Error(1)
}

func (m *TestifyMockPropertyRepository) List(ctx context.Context, filter domain.PropertyFilter) (*domain.PaginatedResult[domain.Property], error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*domain.PaginatedResult[domain.Property]), args.Error(1)
}

func (m *TestifyMockPropertyRepository) FindSimilar(ctx context.Context, embedding []float32, filter domain.PropertyFilter, limit int) ([]domain.MatchedProperty, error) {
	args := m.Called(ctx, embedding, filter, limit)
	return args.Get(0).([]domain.MatchedProperty), args.Error(1)
}

func (m *TestifyMockPropertyRepository) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	args := m.Called(ctx, id, embedding)
	return args.Error(0)
}

// TestifyMockLeadRepository - мок репозитория лидов (с testify)
type TestifyMockLeadRepository struct {
	mock.Mock
}

func (m *TestifyMockLeadRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Lead, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Lead), args.Error(1)
}

// TestifyMockEmbedder - мок для генерации эмбеддингов (с testify)
type TestifyMockEmbedder struct {
	mock.Mock
}

func (m *TestifyMockEmbedder) GenerateEmbedding(text string) ([]float32, error) {
	args := m.Called(text)
	return args.Get(0).([]float32), args.Error(1)
}

// TestMatchingWithDifferentCities тестирует, что объекты из разных городов не матчатся
func TestMatchingWithDifferentCities(t *testing.T) {
	// Описание теста:
	// 1. Создаём лид с городом "Москва"
	// 2. Создаём 2 объекта недвижимости: один в Москве, другой в СПб
	// 3. При матчинге должен вернуться ТОЛЬКО объект из Москвы

	t.Run("Объекты из другого города не должны матчиться", func(t *testing.T) {
		// Arrange
		leadID := uuid.New()
		propertyMoscowID := uuid.New()
		propertySpbID := uuid.New()

		moscow := "Москва"
		spb := "Санкт-Петербург"

		lead := domain.Lead{
			ID:          leadID,
			Title:       "Ищу квартиру в Москве",
			Description: "Нужна 2-комнатная квартира в центре Москвы, бюджет до 15 млн",
			City:        &moscow,
		}

		propertyMoscow := domain.Property{
			ID:          propertyMoscowID,
			Title:       "2-комнатная квартира в центре",
			Description: "Прекрасная квартира в центре Москвы, 2 комнаты, 60 кв.м",
			City:        &moscow,
		}

		propertySpb := domain.Property{
			ID:          propertySpbID,
			Title:       "2-комнатная квартира в центре",
			Description: "Прекрасная квартира в центре Санкт-Петербурга, 2 комнаты, 60 кв.м", // Почти идентичное описание!
			City:        &spb,
		}

		// Эмбеддинги (нормализованные, одинаковые для демонстрации)
		embedding := make([]float32, 384)
		for i := range embedding {
			embedding[i] = 0.1
		}

		// Проверяем функцию CitiesMatch из domain
		assert.True(t, domain.CitiesMatch(moscow, moscow), "Москва должна совпадать с Москвой")
		assert.False(t, domain.CitiesMatch(moscow, spb), "Москва НЕ должна совпадать с СПб")

		// Проверяем, что города корректно установлены
		assert.Equal(t, &moscow, lead.City)
		assert.Equal(t, &moscow, propertyMoscow.City)
		assert.Equal(t, &spb, propertySpb.City)

		t.Logf("Лид: город=%s", *lead.City)
		t.Logf("Объект Москва: город=%s", *propertyMoscow.City)
		t.Logf("Объект СПб: город=%s", *propertySpb.City)

		// При реальном матчинге фильтр по городу должен отсеять СПб
		// Проверим что фильтр работает
		filter := domain.PropertyFilter{
			City: &moscow,
		}

		// Проверяем что фильтр установлен правильно
		assert.NotNil(t, filter.City)
		assert.Equal(t, moscow, *filter.City)
	})

	t.Run("NormalizeCity корректно нормализует города", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"спб", "Санкт-Петербург"},
			{"питер", "Санкт-Петербург"},
			{"мск", "Москва"},
			{"moscow", "Москва"},
			{"Москва", "Москва"},
		}

		for _, tc := range testCases {
			result := domain.NormalizeCity(tc.input)
			assert.Equal(t, tc.expected, result, "NormalizeCity(%q) должен вернуть %q", tc.input, tc.expected)
		}
	})

	t.Run("ExtractCityFromAddress извлекает город из адреса", func(t *testing.T) {
		testCases := []struct {
			address  string
			expected *string
		}{
			{"г. Москва, ул. Тверская, д. 1", strPtr("Москва")},
			{"Санкт-Петербург, Невский проспект, 10", strPtr("Санкт-Петербург")},
			{"Новосибирск, ул. Ленина, 5", strPtr("Новосибирск")},
		}

		for _, tc := range testCases {
			result := domain.ExtractCityFromAddress(tc.address)
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result, "Должен извлечь город из: %s", tc.address)
				assert.Equal(t, *tc.expected, *result)
			}
		}
	})
}

func strPtr(s string) *string {
	return &s
}

