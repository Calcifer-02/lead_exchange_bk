package propertygrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateProperty — создание нового объекта недвижимости.
func (s *propertyServer) CreateProperty(ctx context.Context, in *pb.CreatePropertyRequest) (*pb.PropertyResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	property := domain.Property{
		Title:         in.Title,
		Description:   in.Description,
		Address:       in.Address,
		PropertyType:  protoPropertyTypeToDomain(in.PropertyType),
		Status:        domain.PropertyStatusNew,
		OwnerUserID:   userID,
		CreatedUserID: userID,
	}

	if in.Area != nil {
		property.Area = in.Area
	}
	if in.Price != nil {
		property.Price = in.Price
	}
	if in.Rooms != nil {
		property.Rooms = in.Rooms
	}

	id, err := s.propertyService.CreateProperty(ctx, property)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create property: %v", err))
	}

	property.ID = id
	return &pb.PropertyResponse{Property: propertyDomainToProto(property)}, nil
}

