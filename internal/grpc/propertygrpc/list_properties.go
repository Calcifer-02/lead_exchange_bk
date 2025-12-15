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

// ListProperties — получение списка объектов недвижимости по фильтру.
func (s *propertyServer) ListProperties(ctx context.Context, in *pb.ListPropertiesRequest) (*pb.ListPropertiesResponse, error) {
	filter := domain.PropertyFilter{}

	if in.Filter != nil {
		if in.Filter.Status != nil {
			statusStr := protoPropertyStatusToDomain(*in.Filter.Status)
			filter.Status = &statusStr
		}
		if in.Filter.OwnerUserId != nil {
			id, err := uuid.Parse(*in.Filter.OwnerUserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid owner_user_id")
			}
			filter.OwnerUserID = &id
		}
		if in.Filter.CreatedUserId != nil {
			id, err := uuid.Parse(*in.Filter.CreatedUserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid created_user_id")
			}
			filter.CreatedUserID = &id
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

	properties, err := s.propertyService.ListProperties(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list properties: %v", err))
	}

	resp := &pb.ListPropertiesResponse{}
	for _, p := range properties {
		resp.Properties = append(resp.Properties, propertyDomainToProto(p))
	}
	return resp, nil
}

