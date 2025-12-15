package leadgrpc

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	"lead_exchange/internal/domain"
	"lead_exchange/internal/middleware"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateLead — создание нового лида.
func (s *leadServer) CreateLead(ctx context.Context, in *pb.CreateLeadRequest) (*pb.LeadResponse, error) {
	if err := in.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userID, ok := middleware.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}

	lead := domain.Lead{
		Title:         in.Title,
		Description:   in.Description,
		Requirement:   in.Requirement,
		ContactName:   in.ContactName,
		ContactPhone:  in.ContactPhone,
		ContactEmail:  lo.EmptyableToPtr(in.ContactEmail),
		Status:        domain.LeadStatusNew,
		OwnerUserID:   userID,
		CreatedUserID: userID,
	}

	id, err := s.leadService.CreateLead(ctx, lead)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create lead: %v", err))
	}

	lead.ID = id
	return &pb.LeadResponse{Lead: leadDomainToProto(lead)}, nil
}
