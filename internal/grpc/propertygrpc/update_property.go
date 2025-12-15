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

// UpdateProperty — частичное обновление данных объекта недвижимости.
func (s *propertyServer) UpdateProperty(ctx context.Context, in *pb.UpdatePropertyRequest) (*pb.PropertyResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := uuid.Parse(in.GetPropertyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid property_id format")
	}

	filter := domain.PropertyFilter{
		Title:       in.Title,
		Description: in.Description,
		Address:     in.Address,
	}

	if in.PropertyType != nil {
		propertyTypeStr := protoPropertyTypeToDomain(*in.PropertyType)
		filter.PropertyType = &propertyTypeStr
	}

	if in.Area != nil {
		filter.Area = in.Area
	}

	if in.Price != nil {
		filter.Price = in.Price
	}

	if in.Rooms != nil {
		filter.Rooms = in.Rooms
	}

	if in.Status != nil {
		statusStr := protoPropertyStatusToDomain(*in.Status)
		filter.Status = &statusStr
	}

	if in.OwnerUserId != nil {
		ownerID, err := uuid.Parse(*in.OwnerUserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid owner_user_id")
		}
		filter.OwnerUserID = &ownerID
	}

	updated, err := s.propertyService.UpdateProperty(ctx, id, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update property: %v", err))
	}

	return &pb.PropertyResponse{Property: propertyDomainToProto(updated)}, nil
}

