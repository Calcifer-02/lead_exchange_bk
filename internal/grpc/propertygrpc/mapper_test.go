package propertygrpc

import (
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPropertyDomainToProto(t *testing.T) {
	propertyID := uuid.New()
	ownerID := uuid.New()
	createdUserID := uuid.New()
	area := 75.5
	price := int64(5000000)
	rooms := int32(3)
	city := "Москва"

	property := domain.Property{
		ID:            propertyID,
		Title:         "Тестовая квартира",
		Description:   "Описание квартиры",
		Address:       "ул. Тестовая, д. 1",
		City:          &city,
		PropertyType:  domain.PropertyTypeApartment,
		Area:          &area,
		Price:         &price,
		Rooms:         &rooms,
		Status:        domain.PropertyStatusPublished,
		OwnerUserID:   ownerID,
		CreatedUserID: createdUserID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	proto := propertyDomainToProto(property)

	if proto.PropertyId != propertyID.String() {
		t.Errorf("expected PropertyId %s, got %s", propertyID.String(), proto.PropertyId)
	}
	if proto.Title != "Тестовая квартира" {
		t.Errorf("expected Title 'Тестовая квартира', got %s", proto.Title)
	}
	if proto.Description != "Описание квартиры" {
		t.Errorf("expected Description 'Описание квартиры', got %s", proto.Description)
	}
	if proto.Address != "ул. Тестовая, д. 1" {
		t.Errorf("expected Address 'ул. Тестовая, д. 1', got %s", proto.Address)
	}
	if proto.City == nil || *proto.City != "Москва" {
		t.Errorf("expected City 'Москва', got %v", proto.City)
	}
	if proto.PropertyType != pb.PropertyType_PROPERTY_TYPE_APARTMENT {
		t.Errorf("expected PropertyType APARTMENT, got %v", proto.PropertyType)
	}
	if proto.Area == nil || *proto.Area != 75.5 {
		t.Errorf("expected Area 75.5, got %v", proto.Area)
	}
	if proto.Price == nil || *proto.Price != 5000000 {
		t.Errorf("expected Price 5000000, got %v", proto.Price)
	}
	if proto.Rooms == nil || *proto.Rooms != 3 {
		t.Errorf("expected Rooms 3, got %v", proto.Rooms)
	}
	if proto.Status != pb.PropertyStatus_PROPERTY_STATUS_PUBLISHED {
		t.Errorf("expected Status PUBLISHED, got %v", proto.Status)
	}
}

func TestPropertyTypeDomainToProto(t *testing.T) {
	tests := []struct {
		domain domain.PropertyType
		proto  pb.PropertyType
	}{
		{domain.PropertyTypeApartment, pb.PropertyType_PROPERTY_TYPE_APARTMENT},
		{domain.PropertyTypeHouse, pb.PropertyType_PROPERTY_TYPE_HOUSE},
		{domain.PropertyTypeCommercial, pb.PropertyType_PROPERTY_TYPE_COMMERCIAL},
		{domain.PropertyTypeLand, pb.PropertyType_PROPERTY_TYPE_LAND},
		{domain.PropertyTypeUnspecified, pb.PropertyType_PROPERTY_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := propertyTypeDomainToProto(tt.domain)
		if result != tt.proto {
			t.Errorf("propertyTypeDomainToProto(%v) = %v, want %v", tt.domain, result, tt.proto)
		}
	}
}

func TestProtoPropertyTypeToDomain(t *testing.T) {
	tests := []struct {
		proto  pb.PropertyType
		domain domain.PropertyType
	}{
		{pb.PropertyType_PROPERTY_TYPE_APARTMENT, domain.PropertyTypeApartment},
		{pb.PropertyType_PROPERTY_TYPE_HOUSE, domain.PropertyTypeHouse},
		{pb.PropertyType_PROPERTY_TYPE_COMMERCIAL, domain.PropertyTypeCommercial},
		{pb.PropertyType_PROPERTY_TYPE_LAND, domain.PropertyTypeLand},
		{pb.PropertyType_PROPERTY_TYPE_UNSPECIFIED, domain.PropertyTypeUnspecified},
	}

	for _, tt := range tests {
		result := protoPropertyTypeToDomain(tt.proto)
		if result != tt.domain {
			t.Errorf("protoPropertyTypeToDomain(%v) = %v, want %v", tt.proto, result, tt.domain)
		}
	}
}

func TestPropertyStatusDomainToProto(t *testing.T) {
	tests := []struct {
		domain domain.PropertyStatus
		proto  pb.PropertyStatus
	}{
		{domain.PropertyStatusNew, pb.PropertyStatus_PROPERTY_STATUS_NEW},
		{domain.PropertyStatusPublished, pb.PropertyStatus_PROPERTY_STATUS_PUBLISHED},
		{domain.PropertyStatusSold, pb.PropertyStatus_PROPERTY_STATUS_SOLD},
		{domain.PropertyStatusDeleted, pb.PropertyStatus_PROPERTY_STATUS_DELETED},
		{domain.PropertyStatusUnspecified, pb.PropertyStatus_PROPERTY_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := propertyStatusDomainToProto(tt.domain)
		if result != tt.proto {
			t.Errorf("propertyStatusDomainToProto(%v) = %v, want %v", tt.domain, result, tt.proto)
		}
	}
}

func TestMatchedPropertyToProto(t *testing.T) {
	propertyID := uuid.New()
	ownerID := uuid.New()
	createdUserID := uuid.New()

	totalScore := 0.85
	priceScore := 0.9
	districtScore := 0.8
	roomsScore := 1.0
	areaScore := 0.7
	semanticScore := 0.75
	explanation := "Отличное соответствие по цене и району"

	matched := domain.MatchedProperty{
		Property: domain.Property{
			ID:            propertyID,
			Title:         "Квартира",
			Description:   "Описание",
			Address:       "Адрес",
			PropertyType:  domain.PropertyTypeApartment,
			Status:        domain.PropertyStatusPublished,
			OwnerUserID:   ownerID,
			CreatedUserID: createdUserID,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		Similarity:       0.92,
		TotalScore:       &totalScore,
		PriceScore:       &priceScore,
		DistrictScore:    &districtScore,
		RoomsScore:       &roomsScore,
		AreaScore:        &areaScore,
		SemanticScore:    &semanticScore,
		MatchExplanation: &explanation,
	}

	proto := matchedPropertyToProto(matched)

	if proto.Similarity != 0.92 {
		t.Errorf("expected Similarity 0.92, got %f", proto.Similarity)
	}
	if proto.TotalScore == nil || *proto.TotalScore != 0.85 {
		t.Errorf("expected TotalScore 0.85, got %v", proto.TotalScore)
	}
	if proto.PriceScore == nil || *proto.PriceScore != 0.9 {
		t.Errorf("expected PriceScore 0.9, got %v", proto.PriceScore)
	}
	if proto.DistrictScore == nil || *proto.DistrictScore != 0.8 {
		t.Errorf("expected DistrictScore 0.8, got %v", proto.DistrictScore)
	}
	if proto.RoomsScore == nil || *proto.RoomsScore != 1.0 {
		t.Errorf("expected RoomsScore 1.0, got %v", proto.RoomsScore)
	}
	if proto.AreaScore == nil || *proto.AreaScore != 0.7 {
		t.Errorf("expected AreaScore 0.7, got %v", proto.AreaScore)
	}
	if proto.SemanticScore == nil || *proto.SemanticScore != 0.75 {
		t.Errorf("expected SemanticScore 0.75, got %v", proto.SemanticScore)
	}
	if proto.MatchExplanation == nil || *proto.MatchExplanation != explanation {
		t.Errorf("expected MatchExplanation '%s', got %v", explanation, proto.MatchExplanation)
	}
}

func TestMatchedPropertyToProto_NilScores(t *testing.T) {
	propertyID := uuid.New()
	ownerID := uuid.New()
	createdUserID := uuid.New()

	matched := domain.MatchedProperty{
		Property: domain.Property{
			ID:            propertyID,
			Title:         "Квартира",
			Description:   "Описание",
			Address:       "Адрес",
			PropertyType:  domain.PropertyTypeApartment,
			Status:        domain.PropertyStatusPublished,
			OwnerUserID:   ownerID,
			CreatedUserID: createdUserID,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		Similarity: 0.85,
		// Все scores nil
	}

	proto := matchedPropertyToProto(matched)

	if proto.Similarity != 0.85 {
		t.Errorf("expected Similarity 0.85, got %f", proto.Similarity)
	}
	if proto.TotalScore != nil {
		t.Errorf("expected TotalScore to be nil, got %v", proto.TotalScore)
	}
	if proto.PriceScore != nil {
		t.Errorf("expected PriceScore to be nil, got %v", proto.PriceScore)
	}
	if proto.DistrictScore != nil {
		t.Errorf("expected DistrictScore to be nil, got %v", proto.DistrictScore)
	}
}

func TestParseUUID(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	result, err := parseUUID(validUUID)
	if err != nil {
		t.Errorf("unexpected error parsing valid UUID: %v", err)
	}
	if result.String() != validUUID {
		t.Errorf("expected %s, got %s", validUUID, result.String())
	}

	_, err = parseUUID("invalid-uuid")
	if err == nil {
		t.Error("expected error parsing invalid UUID")
	}
}

