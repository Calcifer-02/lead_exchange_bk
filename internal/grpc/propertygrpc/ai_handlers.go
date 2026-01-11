package propertygrpc

import (
	"context"
	"fmt"

	"lead_exchange/internal/domain"
	"lead_exchange/internal/lib/jsonld"
	"lead_exchange/internal/lib/llm"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MatchPropertiesAdvanced — расширенный поиск с гибридным поиском и реранкером.
func (s *serverAPI) MatchPropertiesAdvanced(ctx context.Context, in *pb.MatchPropertiesAdvancedRequest) (*pb.MatchPropertiesResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	leadID, err := uuid.Parse(in.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id format")
	}

	// Формируем фильтр
	filter := domain.PropertyFilter{}
	if in.Filter != nil {
		if in.Filter.Status != nil {
			statusStr := protoPropertyStatusToDomain(*in.Filter.Status)
			filter.Status = &statusStr
		}
		if in.Filter.PropertyType != nil {
			propertyTypeStr := protoPropertyTypeToDomain(*in.Filter.PropertyType)
			filter.PropertyType = &propertyTypeStr
		}
		if in.Filter.MinRooms != nil {
			filter.MinRooms = in.Filter.MinRooms
		}
		if in.Filter.MaxRooms != nil {
			filter.MaxRooms = in.Filter.MaxRooms
		}
		if in.Filter.MinPrice != nil {
			filter.MinPrice = in.Filter.MinPrice
		}
		if in.Filter.MaxPrice != nil {
			filter.MaxPrice = in.Filter.MaxPrice
		}
		if in.Filter.City != nil {
			filter.City = in.Filter.City
		}
	}

	limit := 10
	if in.Limit != nil && *in.Limit > 0 {
		limit = int(*in.Limit)
	}

	// Используем расширенный поиск с поддержкой AI-функций
	matches, err := s.propertyService.MatchPropertiesAdvanced(ctx, leadID, filter, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to match properties: %v", err))
	}

	resp := &pb.MatchPropertiesResponse{}
	for _, match := range matches {
		pbMatch := matchedPropertyToProto(match)
		resp.Matches = append(resp.Matches, pbMatch)
	}

	return resp, nil
}

// GetPropertyJSONLD — получить JSON-LD разметку объекта недвижимости (schema.org).
func (s *serverAPI) GetPropertyJSONLD(ctx context.Context, in *pb.GetPropertyJSONLDRequest) (*pb.GetPropertyJSONLDResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	propertyID, err := uuid.Parse(in.GetPropertyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid property_id format")
	}

	// Получаем объект недвижимости
	property, err := s.propertyService.GetProperty(ctx, propertyID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("property not found: %v", err))
	}

	// Определяем base URL
	baseURL := "https://api.leadexchange.ru"
	if in.BaseUrl != nil && *in.BaseUrl != "" {
		baseURL = *in.BaseUrl
	}

	// Генерируем JSON-LD
	generator := jsonld.NewGenerator()
	jsonldData, err := generator.GeneratePropertyJSONLDBytes(property, baseURL)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to generate JSON-LD: %v", err))
	}

	return &pb.GetPropertyJSONLDResponse{
		JsonldData: jsonldData,
	}, nil
}

// GenerateListingContent — AI-генерация заголовка и описания для объекта.
func (s *serverAPI) GenerateListingContent(ctx context.Context, in *pb.GenerateListingContentRequest) (*pb.GenerateListingContentResponse, error) {
	// Проверяем, доступен ли LLM клиент
	if s.llmClient == nil || !s.llmClient.IsEnabled() {
		return nil, status.Error(codes.Unavailable, "LLM service is not available")
	}

	// Получаем данные из существующего объекта, если указан property_id
	var existingTitle, existingDescription string
	if in.PropertyId != nil && *in.PropertyId != "" {
		propertyID, err := uuid.Parse(*in.PropertyId)
		if err == nil {
			property, err := s.propertyService.GetProperty(ctx, propertyID)
			if err == nil {
				existingTitle = property.Title
				existingDescription = property.Description
			}
		}
	}

	// Если указаны existing_title/existing_description в запросе, используем их
	if in.ExistingTitle != nil && *in.ExistingTitle != "" {
		existingTitle = *in.ExistingTitle
	}
	if in.ExistingDescription != nil && *in.ExistingDescription != "" {
		existingDescription = *in.ExistingDescription
	}

	// Формируем запрос к LLM
	req := llm.GenerateListingRequest{
		PropertyType:        stringOrEmpty(in.PropertyType),
		Address:             stringOrEmpty(in.Address),
		City:                stringOrEmpty(in.City),
		ExistingTitle:       existingTitle,
		ExistingDescription: existingDescription,
		Features:            in.Features,
	}

	if in.Price != nil {
		req.Price = in.Price
	}
	if in.Rooms != nil {
		req.Rooms = in.Rooms
	}
	if in.Area != nil {
		req.Area = in.Area
	}

	// Генерируем контент
	resp, err := s.llmClient.GenerateListingContent(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to generate content: %v", err))
	}

	return &pb.GenerateListingContentResponse{
		Title:       resp.Title,
		Description: resp.Description,
		Keywords:    resp.Keywords,
		Confidence:  resp.Confidence,
	}, nil
}

// AnalyzePropertyImages — анализ изображений объекта недвижимости.
// Пока не реализовано, так как изображения не хранятся.
func (s *serverAPI) AnalyzePropertyImages(ctx context.Context, in *pb.AnalyzePropertyImagesRequest) (*pb.AnalyzePropertyImagesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "image analysis is not yet available - images are not stored in the system")
}

// stringOrEmpty возвращает строку из optional string или пустую строку.
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

