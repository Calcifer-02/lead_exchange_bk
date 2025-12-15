package propertygrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MatchProperties — поиск подходящих объектов недвижимости для лида.
// Поддерживает взвешенное ранжирование через metadata:
// - x-weights-preset: ID пресета (balanced, budget_first, location_first, family, semantic)
// - x-use-weighted-ranking: "true" для включения
// - x-criteria-json: JSON с SoftCriteria
func (s *propertyServer) MatchProperties(ctx context.Context, in *pb.MatchPropertiesRequest) (*pb.MatchPropertiesResponse, error) {
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
	}

	limit := 10
	if in.Limit != nil && *in.Limit > 0 {
		limit = int(*in.Limit)
	}

	// Взвешенное ранжирование включено по умолчанию с пресетом "balanced"
	defaultWeights := domain.DefaultWeights()
	weights := &defaultWeights
	var criteria *domain.SoftCriteria
	useWeightedRanking := true

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// Переопределяем пресет весов если указан
		if vals := md.Get("x-weights-preset"); len(vals) > 0 {
			if preset := domain.GetWeightPresetByID(vals[0]); preset != nil {
				weights = &preset.Weights
			}
		}
		// Можно отключить взвешенное ранжирование явно
		if vals := md.Get("x-use-weighted-ranking"); len(vals) > 0 && vals[0] == "false" {
			useWeightedRanking = false
		}
		// Парсим критерии из JSON
		if vals := md.Get("x-criteria-json"); len(vals) > 0 {
			var c domain.SoftCriteria
			if err := json.Unmarshal([]byte(vals[0]), &c); err == nil {
				criteria = &c
			}
		}
	}

	matches, err := s.propertyService.MatchPropertiesWeighted(ctx, leadID, filter, limit, weights, criteria, useWeightedRanking)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to match properties: %v", err))
	}

	resp := &pb.MatchPropertiesResponse{}
	for _, match := range matches {
		pbMatch := &pb.MatchedProperty{
			Property:   propertyDomainToProto(match.Property),
			Similarity: match.Similarity,
		}
		// TODO: После обновления proto добавить TotalScore, PriceScore и др.
		resp.Matches = append(resp.Matches, pbMatch)
	}

	return resp, nil
}
