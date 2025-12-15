package dealgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AcceptDeal — принятие сделки покупателем.
func (s *dealServer) AcceptDeal(ctx context.Context, in *pb.AcceptDealRequest) (*pb.DealResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.checkUserStatus(ctx); err != nil {
		return nil, err
	}

	dealID, err := uuid.Parse(in.DealId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid deal_id: %v", err))
	}

	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	deal, err := s.dealService.AcceptDeal(ctx, dealID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to accept deal: %v", err))
	}

	return &pb.DealResponse{Deal: dealDomainToProto(deal)}, nil
}
