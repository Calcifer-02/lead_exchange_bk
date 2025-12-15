package leadgrpc

import (
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/samber/lo"
)

func leadDomainToProto(l domain.Lead) *pb.Lead {
	return &pb.Lead{
		LeadId:        l.ID.String(),
		Title:         l.Title,
		Description:   l.Description,
		Requirement:   l.Requirement,
		ContactName:   l.ContactName,
		ContactPhone:  l.ContactPhone,
		ContactEmail:  lo.FromPtr(l.ContactEmail),
		Status:        leadStatusDomainToProto(l.Status),
		OwnerUserId:   l.OwnerUserID.String(),
		CreatedUserId: l.CreatedUserID.String(),
		CreatedAt:     l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func leadStatusDomainToProto(s domain.LeadStatus) pb.LeadStatus {
	switch s {
	case domain.LeadStatusNew:
		return pb.LeadStatus_LEAD_STATUS_NEW
	case domain.LeadStatusPublished:
		return pb.LeadStatus_LEAD_STATUS_PUBLISHED
	case domain.LeadStatusPurchased:
		return pb.LeadStatus_LEAD_STATUS_PURCHASED
	case domain.LeadStatusDeleted:
		return pb.LeadStatus_LEAD_STATUS_DELETED
	default:
		return pb.LeadStatus_LEAD_STATUS_UNSPECIFIED
	}
}

func protoLeadStatusToDomain(s pb.LeadStatus) domain.LeadStatus {
	switch s {
	case pb.LeadStatus_LEAD_STATUS_NEW:
		return domain.LeadStatusNew
	case pb.LeadStatus_LEAD_STATUS_PUBLISHED:
		return domain.LeadStatusPublished
	case pb.LeadStatus_LEAD_STATUS_PURCHASED:
		return domain.LeadStatusPurchased
	case pb.LeadStatus_LEAD_STATUS_DELETED:
		return domain.LeadStatusDeleted
	default:
		return domain.LeadStatusUnspecified
	}
}
