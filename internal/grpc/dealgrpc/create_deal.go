package dealgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateDeal — создание новой сделки.
func (s *dealServer) CreateDeal(ctx context.Context, in *pb.CreateDealRequest) (*pb.DealResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.checkUserStatus(ctx); err != nil {
		return nil, err
	}

	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	leadID, err := parseUUID(in.LeadId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid lead_id: %v", err))
	}

	deal := domain.Deal{
		LeadID:       leadID,
		SellerUserID: userID,
		Price:        in.Price,
		Status:       domain.DealStatusPending,
	}

	id, err := s.dealService.CreateDeal(ctx, deal)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create deal: %v", err))
	}

	deal.ID = id
	return &pb.DealResponse{Deal: dealDomainToProto(deal)}, nil
}
