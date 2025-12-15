package leadgrpc

import (
	"context"
	"fmt"
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListLeads — получение списка лидов по фильтру.
func (s *leadServer) ListLeads(ctx context.Context, in *pb.ListLeadsRequest) (*pb.ListLeadsResponse, error) {
	filter := domain.LeadFilter{}

	if in.Filter != nil {
		if in.Filter.Status != nil {
			statusStr := protoLeadStatusToDomain(*in.Filter.Status)
			filter.Status = &statusStr
		}
		if in.Filter.OwnerUserId != nil {
			id, err := uuid.Parse(*in.Filter.OwnerUserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid owner_user_id")
			}
			filter.OwnerUserID = &id
		}
		if in.Filter.CreatedUserId != nil {
			id, err := uuid.Parse(*in.Filter.CreatedUserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid created_user_id")
			}
			filter.CreatedUserID = &id
		}
	}

	leads, err := s.leadService.ListLeads(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list leads: %v", err))
	}

	resp := &pb.ListLeadsResponse{}
	for _, l := range leads {
		resp.Leads = append(resp.Leads, leadDomainToProto(l))
	}
	return resp, nil
}
