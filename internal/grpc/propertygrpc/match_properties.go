package propertygrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MatchProperties — поиск подходящих объектов недвижимости для лида.
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

	matches, err := s.propertyService.MatchProperties(ctx, leadID, filter, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to match properties: %v", err))
	}

	resp := &pb.MatchPropertiesResponse{}
	for _, match := range matches {
		resp.Matches = append(resp.Matches, &pb.MatchedProperty{
			Property:   propertyDomainToProto(match.Property),
			Similarity: match.Similarity,
		})
	}

	return resp, nil
}

