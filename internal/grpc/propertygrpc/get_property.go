package propertygrpc

import (
	"context"
	"fmt"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetProperty — получение информации об объекте недвижимости по ID.
func (s *propertyServer) GetProperty(ctx context.Context, in *pb.GetPropertyRequest) (*pb.PropertyResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := uuid.Parse(in.GetPropertyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid property_id format")
	}

	property, err := s.propertyService.GetProperty(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get property: %v", err))
	}

	return &pb.PropertyResponse{Property: propertyDomainToProto(property)}, nil
}

