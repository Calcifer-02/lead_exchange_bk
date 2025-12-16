package domain

import (
	"regexp"
	"strings"
)

// KnownCities — список известных городов России для автоматического определения.
var KnownCities = []string{
	"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань",
	"Нижний Новгород", "Челябинск", "Самара", "Омск", "Ростов-на-Дону",
	"Уфа", "Красноярск", "Воронеж", "Пермь", "Волгоград",
	"Краснодар", "Саратов", "Тюмень", "Тольятти", "Ижевск",
	"Барнаул", "Ульяновск", "Иркутск", "Хабаровск", "Ярославль",
	"Владивосток", "Махачкала", "Томск", "Оренбург", "Кемерово",
	"Новокузнецк", "Рязань", "Астрахань", "Набережные Челны", "Пенза",
	"Липецк", "Киров", "Чебоксары", "Тула", "Калининград",
	"Сочи", "Севастополь", "Симферополь",
	// Английские варианты
	"Moscow", "Saint Petersburg", "St. Petersburg",
}

// ExtractCityFromAddress пытается извлечь город из адреса.
// Возвращает nil, если город не удалось определить.
func ExtractCityFromAddress(address string) *string {
	if address == "" {
		return nil
	}

	addressLower := strings.ToLower(address)

	// Ищем известные города
	for _, city := range KnownCities {
		if strings.Contains(addressLower, strings.ToLower(city)) {
			return &city
		}
	}

	// Пытаемся извлечь город по шаблону "г. Название" или "город Название"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)г\.\s*([А-Яа-яЁё-]+)`),
		regexp.MustCompile(`(?i)город\s+([А-Яа-яЁё-]+)`),
		regexp.MustCompile(`(?i)^([А-Яа-яЁё-]+),`), // Город в начале адреса до запятой
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(address); len(matches) > 1 {
			city := strings.TrimSpace(matches[1])
			if len(city) > 2 { // Минимум 3 символа для названия города
				return &city
			}
		}
	}

	return nil
}

// NormalizeCity приводит название города к единому виду.
func NormalizeCity(city string) string {
	city = strings.TrimSpace(city)

	// Приводим к стандартным названиям
	normalizations := map[string]string{
		"спб":               "Санкт-Петербург",
		"питер":             "Санкт-Петербург",
		"санкт петербург":   "Санкт-Петербург",
		"saint petersburg":  "Санкт-Петербург",
		"st. petersburg":    "Санкт-Петербург",
		"мск":               "Москва",
		"moscow":            "Москва",
		"нск":               "Новосибирск",
		"екб":               "Екатеринбург",
		"ростов":            "Ростов-на-Дону",
		"нижний":            "Нижний Новгород",
	}

	cityLower := strings.ToLower(city)
	if normalized, ok := normalizations[cityLower]; ok {
		return normalized
	}

	// Делаем первую букву заглавной
	if len(city) > 0 {
		runes := []rune(city)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		return string(runes)
	}

	return city
}

// CitiesMatch проверяет, совпадают ли два города (с учётом нормализации).
func CitiesMatch(city1, city2 string) bool {
	if city1 == "" || city2 == "" {
		return false
	}
	return strings.EqualFold(NormalizeCity(city1), NormalizeCity(city2))
}

