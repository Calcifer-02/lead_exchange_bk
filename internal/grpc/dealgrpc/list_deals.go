package dealgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListDeals — получение списка сделок по фильтру.
func (s *dealServer) ListDeals(ctx context.Context, in *pb.ListDealsRequest) (*pb.ListDealsResponse, error) {
	if err := s.checkUserStatus(ctx); err != nil {
		return nil, err
	}

	filter := domain.DealFilter{}

	if in.Filter != nil {
		if in.Filter.LeadId != nil && *in.Filter.LeadId != "" {
			leadID, err := uuid.Parse(*in.Filter.LeadId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid lead_id: %v", err))
			}
			filter.LeadID = &leadID
		}

		if in.Filter.SellerUserId != nil && *in.Filter.SellerUserId != "" {
			sellerID, err := uuid.Parse(*in.Filter.SellerUserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid seller_user_id: %v", err))
			}
			filter.SellerUserID = &sellerID
		}

		if in.Filter.BuyerUserId != nil && *in.Filter.BuyerUserId != "" {
			buyerID, err := uuid.Parse(*in.Filter.BuyerUserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid buyer_user_id: %v", err))
			}
			filter.BuyerUserID = &buyerID
		}

		if in.Filter.Status != nil {
			status := protoDealStatusToDomain(*in.Filter.Status)
			filter.Status = &status
		}

		if in.Filter.MinPrice != nil {
			filter.MinPrice = in.Filter.MinPrice
		}

		if in.Filter.MaxPrice != nil {
			filter.MaxPrice = in.Filter.MaxPrice
		}
	}

	deals, err := s.dealService.ListDeals(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list deals: %v", err))
	}

	protoDeals := make([]*pb.Deal, len(deals))
	for i, deal := range deals {
		protoDeals[i] = dealDomainToProto(deal)
	}

	return &pb.ListDealsResponse{Deals: protoDeals}, nil
}
