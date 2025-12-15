package dealgrpc

import (
	"context"
	"fmt"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDeal — получение сделки по ID.
func (s *dealServer) GetDeal(ctx context.Context, in *pb.GetDealRequest) (*pb.DealResponse, error) {
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

	deal, err := s.dealService.GetDeal(ctx, dealID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("deal not found: %v", err))
	}

	return &pb.DealResponse{Deal: dealDomainToProto(deal)}, nil
}
