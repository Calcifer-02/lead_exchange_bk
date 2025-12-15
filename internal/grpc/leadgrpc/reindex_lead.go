package leadgrpc

import (
	"context"
	"fmt"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *leadServer) ReindexLead(ctx context.Context, req *pb.ReindexLeadRequest) (*pb.ReindexLeadResponse, error) {
	if req.LeadId == "" {
		return nil, status.Error(codes.InvalidArgument, "lead_id is required")
	}

	id, err := uuid.Parse(req.LeadId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead_id")
	}

	err = s.leadService.ReindexLead(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reindex lead: %v", err)
	}

	return &pb.ReindexLeadResponse{
		Success: true,
		Message: fmt.Sprintf("Lead %s reindexed successfully", id),
	}, nil
}

