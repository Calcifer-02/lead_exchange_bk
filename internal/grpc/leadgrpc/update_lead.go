package leadgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateLead — частичное обновление данных лида.
func (s *leadServer) UpdateLead(ctx context.Context, in *pb.UpdateLeadRequest) (*pb.LeadResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := uuid.Parse(in.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id format")
	}

	filter := domain.LeadFilter{
		Title:       in.Title,
		Description: in.Description,
		Requirement: lo.EmptyableToPtr(in.Requirement),
	}

	if in.Status != nil {
		statusStr := protoLeadStatusToDomain(*in.Status)
		filter.Status = &statusStr
	}

	if in.OwnerUserId != nil {
		ownerID, err := uuid.Parse(*in.OwnerUserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid owner_user_id")
		}
		filter.OwnerUserID = &ownerID
	}

	updated, err := s.leadService.UpdateLead(ctx, id, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update lead: %v", err))
	}

	return &pb.LeadResponse{Lead: leadDomainToProto(updated)}, nil
}
