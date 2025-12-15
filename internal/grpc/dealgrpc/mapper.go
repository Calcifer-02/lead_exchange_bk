package dealgrpc

import (
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"

	"github.com/google/uuid"
)

func dealDomainToProto(d domain.Deal) *pb.Deal {
	buyerUserID := ""
	if d.BuyerUserID != nil {
		buyerUserID = d.BuyerUserID.String()
	}

	return &pb.Deal{
		DealId:       d.ID.String(),
		LeadId:       d.LeadID.String(),
		SellerUserId: d.SellerUserID.String(),
		BuyerUserId:  buyerUserID,
		Price:        d.Price,
		Status:       dealStatusDomainToProto(d.Status),
		CreatedAt:    d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func dealStatusDomainToProto(s domain.DealStatus) pb.DealStatus {
	switch s {
	case domain.DealStatusPending:
		return pb.DealStatus_DEAL_STATUS_PENDING
	case domain.DealStatusAccepted:
		return pb.DealStatus_DEAL_STATUS_ACCEPTED
	case domain.DealStatusCompleted:
		return pb.DealStatus_DEAL_STATUS_COMPLETED
	case domain.DealStatusCancelled:
		return pb.DealStatus_DEAL_STATUS_CANCELLED
	case domain.DealStatusRejected:
		return pb.DealStatus_DEAL_STATUS_REJECTED
	default:
		return pb.DealStatus_DEAL_STATUS_UNSPECIFIED
	}
}

func protoDealStatusToDomain(s pb.DealStatus) domain.DealStatus {
	switch s {
	case pb.DealStatus_DEAL_STATUS_PENDING:
		return domain.DealStatusPending
	case pb.DealStatus_DEAL_STATUS_ACCEPTED:
		return domain.DealStatusAccepted
	case pb.DealStatus_DEAL_STATUS_COMPLETED:
		return domain.DealStatusCompleted
	case pb.DealStatus_DEAL_STATUS_CANCELLED:
		return domain.DealStatusCancelled
	case pb.DealStatus_DEAL_STATUS_REJECTED:
		return domain.DealStatusRejected
	default:
		return domain.DealStatusUnspecified
	}
}

func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}
