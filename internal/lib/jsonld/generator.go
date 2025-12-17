package jsonld

import (
	"encoding/json"
	"fmt"
	"time"

	"lead_exchange/internal/domain"
)

// Generator — генератор JSON-LD разметки для объектов недвижимости.
type Generator struct{}

// NewGenerator создаёт новый генератор JSON-LD.
func NewGenerator() *Generator {
	return &Generator{}
}

// RealEstateListing — JSON-LD структура для листинга недвижимости (schema.org).
type RealEstateListing struct {
	Context         string           `json:"@context"`
	Type            string           `json:"@type"`
	ID              string           `json:"@id,omitempty"`
	Name            string           `json:"name"`
	Description     string           `json:"description,omitempty"`
	URL             string           `json:"url,omitempty"`
	DatePosted      string           `json:"datePosted,omitempty"`
	DateModified    string           `json:"dateModified,omitempty"`

	// Цена
	Offers          *Offer           `json:"offers,omitempty"`

	// Местоположение
	Address         *PostalAddress   `json:"address,omitempty"`
	Geo             *GeoCoordinates  `json:"geo,omitempty"`

	// Характеристики объекта
	FloorSize       *QuantitativeValue `json:"floorSize,omitempty"`
	NumberOfRooms   *int32             `json:"numberOfRooms,omitempty"`

	// Тип недвижимости
	PropertyType    string           `json:"propertyType,omitempty"`

	// Изображения
	Image           []string         `json:"image,omitempty"`

	// Дополнительные свойства
	AdditionalProperty []PropertyValue `json:"additionalProperty,omitempty"`
}

// Offer — предложение (цена) по schema.org.
type Offer struct {
	Type          string `json:"@type"`
	Price         int64  `json:"price,omitempty"`
	PriceCurrency string `json:"priceCurrency"`
	Availability  string `json:"availability,omitempty"`
}

// PostalAddress — почтовый адрес по schema.org.
type PostalAddress struct {
	Type            string `json:"@type"`
	StreetAddress   string `json:"streetAddress,omitempty"`
	AddressLocality string `json:"addressLocality,omitempty"` // Город
	AddressRegion   string `json:"addressRegion,omitempty"`   // Регион
	AddressCountry  string `json:"addressCountry,omitempty"`
}

// GeoCoordinates — географические координаты.
type GeoCoordinates struct {
	Type      string  `json:"@type"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// QuantitativeValue — количественное значение.
type QuantitativeValue struct {
	Type     string  `json:"@type"`
	Value    float64 `json:"value"`
	UnitCode string  `json:"unitCode"` // MTK для м²
	UnitText string  `json:"unitText,omitempty"`
}

// PropertyValue — дополнительное свойство.
type PropertyValue struct {
	Type  string      `json:"@type"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// GeneratePropertyJSONLD генерирует JSON-LD разметку для объекта недвижимости.
func (g *Generator) GeneratePropertyJSONLD(property domain.Property, baseURL string) (*RealEstateListing, error) {
	listing := &RealEstateListing{
		Context:      "https://schema.org",
		Type:         g.mapPropertyType(property.PropertyType),
		ID:           fmt.Sprintf("%s/properties/%s", baseURL, property.ID.String()),
		Name:         property.Title,
		Description:  property.Description,
		URL:          fmt.Sprintf("%s/properties/%s", baseURL, property.ID.String()),
		DatePosted:   property.CreatedAt.Format(time.RFC3339),
		DateModified: property.UpdatedAt.Format(time.RFC3339),
	}

	// Цена
	if property.Price != nil {
		listing.Offers = &Offer{
			Type:          "Offer",
			Price:         *property.Price,
			PriceCurrency: "RUB",
			Availability:  g.mapPropertyStatus(property.Status),
		}
	}

	// Адрес
	listing.Address = &PostalAddress{
		Type:          "PostalAddress",
		StreetAddress: property.Address,
		AddressCountry: "RU",
	}
	if property.City != nil {
		listing.Address.AddressLocality = *property.City
	}

	// Площадь
	if property.Area != nil {
		listing.FloorSize = &QuantitativeValue{
			Type:     "QuantitativeValue",
			Value:    *property.Area,
			UnitCode: "MTK", // Квадратные метры
			UnitText: "м²",
		}
	}

	// Количество комнат
	if property.Rooms != nil {
		listing.NumberOfRooms = property.Rooms
	}

	// Тип недвижимости (текстовый)
	listing.PropertyType = g.mapPropertyTypeText(property.PropertyType)

	return listing, nil
}

// GeneratePropertyJSONLDString генерирует JSON-LD строку.
func (g *Generator) GeneratePropertyJSONLDString(property domain.Property, baseURL string) (string, error) {
	listing, err := g.GeneratePropertyJSONLD(property, baseURL)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(listing, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	return string(data), nil
}

// GeneratePropertyJSONLDBytes генерирует JSON-LD в байтах.
func (g *Generator) GeneratePropertyJSONLDBytes(property domain.Property, baseURL string) ([]byte, error) {
	listing, err := g.GeneratePropertyJSONLD(property, baseURL)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(listing, "", "  ")
}

// AddAdditionalProperties добавляет дополнительные свойства к листингу.
func (g *Generator) AddAdditionalProperties(listing *RealEstateListing, props map[string]interface{}) {
	for name, value := range props {
		listing.AdditionalProperty = append(listing.AdditionalProperty, PropertyValue{
			Type:  "PropertyValue",
			Name:  name,
			Value: value,
		})
	}
}

// AddImages добавляет изображения к листингу.
func (g *Generator) AddImages(listing *RealEstateListing, imageURLs []string) {
	listing.Image = append(listing.Image, imageURLs...)
}

// SetGeoCoordinates устанавливает географические координаты.
func (g *Generator) SetGeoCoordinates(listing *RealEstateListing, lat, lon float64) {
	listing.Geo = &GeoCoordinates{
		Type:      "GeoCoordinates",
		Latitude:  lat,
		Longitude: lon,
	}
}

// mapPropertyType преобразует тип недвижимости в schema.org тип.
func (g *Generator) mapPropertyType(pt domain.PropertyType) string {
	switch pt {
	case domain.PropertyTypeApartment:
		return "Apartment"
	case domain.PropertyTypeHouse:
		return "House"
	case domain.PropertyTypeCommercial:
		return "RealEstateListing" // Общий тип для коммерческой недвижимости
	case domain.PropertyTypeLand:
		return "RealEstateListing" // Общий тип для земельных участков
	default:
		return "RealEstateListing"
	}
}

// mapPropertyTypeText возвращает текстовое описание типа.
func (g *Generator) mapPropertyTypeText(pt domain.PropertyType) string {
	switch pt {
	case domain.PropertyTypeApartment:
		return "Квартира"
	case domain.PropertyTypeHouse:
		return "Дом"
	case domain.PropertyTypeCommercial:
		return "Коммерческая недвижимость"
	case domain.PropertyTypeLand:
		return "Земельный участок"
	default:
		return "Недвижимость"
	}
}

// mapPropertyStatus преобразует статус в schema.org availability.
func (g *Generator) mapPropertyStatus(status domain.PropertyStatus) string {
	switch status {
	case domain.PropertyStatusPublished:
		return "https://schema.org/InStock"
	case domain.PropertyStatusSold:
		return "https://schema.org/SoldOut"
	default:
		return "https://schema.org/InStock"
	}
}

// Apartment — расширенная структура для квартиры.
type Apartment struct {
	RealEstateListing
	NumberOfBedrooms  *int32 `json:"numberOfBedrooms,omitempty"`
	NumberOfBathrooms *int32 `json:"numberOfBathrooms,omitempty"`
	FloorLevel        *int32 `json:"floorLevel,omitempty"`
	YearBuilt         *int32 `json:"yearBuilt,omitempty"`
}

// GenerateApartmentJSONLD генерирует JSON-LD для квартиры с дополнительными полями.
func (g *Generator) GenerateApartmentJSONLD(property domain.Property, baseURL string, bedrooms, bathrooms, floor, yearBuilt *int32) (*Apartment, error) {
	baseListing, err := g.GeneratePropertyJSONLD(property, baseURL)
	if err != nil {
		return nil, err
	}

	baseListing.Type = "Apartment"

	apt := &Apartment{
		RealEstateListing: *baseListing,
		NumberOfBedrooms:  bedrooms,
		NumberOfBathrooms: bathrooms,
		FloorLevel:        floor,
		YearBuilt:         yearBuilt,
	}

	return apt, nil
}

// LeadRequest — JSON-LD структура для запроса на недвижимость.
type LeadRequest struct {
	Context     string       `json:"@context"`
	Type        string       `json:"@type"`
	ID          string       `json:"@id,omitempty"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	DateCreated string       `json:"dateCreated,omitempty"`

	// Желаемые характеристики
	Seeks       *SeeksProperty `json:"seeks,omitempty"`
}

// SeeksProperty — желаемые характеристики недвижимости.
type SeeksProperty struct {
	Type          string             `json:"@type"`
	PropertyType  string             `json:"propertyType,omitempty"`
	Location      *PostalAddress     `json:"location,omitempty"`
	PriceRange    *PriceSpecification `json:"priceSpecification,omitempty"`
	FloorSize     *QuantitativeValue `json:"floorSize,omitempty"`
	NumberOfRooms *int32             `json:"numberOfRooms,omitempty"`
}

// PriceSpecification — спецификация цены.
type PriceSpecification struct {
	Type          string `json:"@type"`
	MinPrice      *int64 `json:"minPrice,omitempty"`
	MaxPrice      *int64 `json:"maxPrice,omitempty"`
	PriceCurrency string `json:"priceCurrency"`
}

// GenerateLeadJSONLD генерирует JSON-LD для лида (запроса на недвижимость).
func (g *Generator) GenerateLeadJSONLD(lead domain.Lead, baseURL string) (*LeadRequest, error) {
	req := &LeadRequest{
		Context:     "https://schema.org",
		Type:        "WantAction", // Действие "хочу"
		ID:          fmt.Sprintf("%s/leads/%s", baseURL, lead.ID.String()),
		Name:        lead.Title,
		Description: lead.Description,
		DateCreated: lead.CreatedAt.Format(time.RFC3339),
	}

	req.Seeks = &SeeksProperty{
		Type: "RealEstateListing",
	}

	if lead.City != nil {
		req.Seeks.Location = &PostalAddress{
			Type:            "PostalAddress",
			AddressLocality: *lead.City,
			AddressCountry:  "RU",
		}
	}

	return req, nil
}

