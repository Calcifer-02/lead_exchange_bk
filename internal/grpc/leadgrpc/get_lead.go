package leadgrpc

import (
	"context"
	"fmt"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetLead — получение информации о лиде по ID.
func (s *leadServer) GetLead(ctx context.Context, in *pb.GetLeadRequest) (*pb.LeadResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := uuid.Parse(in.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id format")
	}

	lead, err := s.leadService.GetLead(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get lead: %v", err))
	}

	return &pb.LeadResponse{Lead: leadDomainToProto(lead)}, nil
}
