package propertygrpc

import (
	"context"
	"fmt"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *propertyServer) ReindexProperty(ctx context.Context, req *pb.ReindexPropertyRequest) (*pb.ReindexPropertyResponse, error) {
	if req.PropertyId == "" {
		return nil, status.Error(codes.InvalidArgument, "property_id is required")
	}

	id, err := uuid.Parse(req.PropertyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid property_id")
	}

	err = s.propertyService.ReindexProperty(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reindex property: %v", err)
	}

	return &pb.ReindexPropertyResponse{
		Success: true,
		Message: fmt.Sprintf("Property %s reindexed successfully", id),
	}, nil
}

