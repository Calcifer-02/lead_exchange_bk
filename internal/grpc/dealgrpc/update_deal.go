package dealgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateDeal — обновление сделки.
func (s *dealServer) UpdateDeal(ctx context.Context, in *pb.UpdateDealRequest) (*pb.DealResponse, error) {
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

	update := domain.DealFilter{}

	if in.Status != nil {
		status := protoDealStatusToDomain(*in.Status)
		update.Status = &status
	}

	if in.Price != nil {
		update.Price = in.Price
	}

	// Проверяем права доступа: только продавец или покупатель могут обновлять сделку
	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	// Получаем текущую сделку для проверки прав
	currentDeal, err := s.dealService.GetDeal(ctx, dealID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("deal not found: %v", err))
	}

	// Проверяем, что пользователь является продавцом или покупателем
	if currentDeal.SellerUserID != userID && (currentDeal.BuyerUserID == nil || *currentDeal.BuyerUserID != userID) {
		return nil, status.Error(codes.PermissionDenied, "only seller or buyer can update deal")
	}

	deal, err := s.dealService.UpdateDeal(ctx, dealID, update)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update deal: %v", err))
	}

	return &pb.DealResponse{Deal: dealDomainToProto(deal)}, nil
}
