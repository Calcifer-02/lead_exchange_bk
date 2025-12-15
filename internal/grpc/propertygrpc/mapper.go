package propertygrpc

import (
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"
)

func propertyDomainToProto(p domain.Property) *pb.Property {
	prop := &pb.Property{
		PropertyId:    p.ID.String(),
		Title:         p.Title,
		Description:   p.Description,
		Address:       p.Address,
		PropertyType:  propertyTypeDomainToProto(p.PropertyType),
		Status:        propertyStatusDomainToProto(p.Status),
		OwnerUserId:   p.OwnerUserID.String(),
		CreatedUserId: p.CreatedUserID.String(),
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if p.Area != nil {
		prop.Area = p.Area
	}
	if p.Price != nil {
		prop.Price = p.Price
	}
	if p.Rooms != nil {
		prop.Rooms = p.Rooms
	}

	return prop
}

func propertyTypeDomainToProto(t domain.PropertyType) pb.PropertyType {
	switch t {
	case domain.PropertyTypeApartment:
		return pb.PropertyType_PROPERTY_TYPE_APARTMENT
	case domain.PropertyTypeHouse:
		return pb.PropertyType_PROPERTY_TYPE_HOUSE
	case domain.PropertyTypeCommercial:
		return pb.PropertyType_PROPERTY_TYPE_COMMERCIAL
	case domain.PropertyTypeLand:
		return pb.PropertyType_PROPERTY_TYPE_LAND
	default:
		return pb.PropertyType_PROPERTY_TYPE_UNSPECIFIED
	}
}

func protoPropertyTypeToDomain(t pb.PropertyType) domain.PropertyType {
	switch t {
	case pb.PropertyType_PROPERTY_TYPE_APARTMENT:
		return domain.PropertyTypeApartment
	case pb.PropertyType_PROPERTY_TYPE_HOUSE:
		return domain.PropertyTypeHouse
	case pb.PropertyType_PROPERTY_TYPE_COMMERCIAL:
		return domain.PropertyTypeCommercial
	case pb.PropertyType_PROPERTY_TYPE_LAND:
		return domain.PropertyTypeLand
	default:
		return domain.PropertyTypeUnspecified
	}
}

func propertyStatusDomainToProto(s domain.PropertyStatus) pb.PropertyStatus {
	switch s {
	case domain.PropertyStatusNew:
		return pb.PropertyStatus_PROPERTY_STATUS_NEW
	case domain.PropertyStatusPublished:
		return pb.PropertyStatus_PROPERTY_STATUS_PUBLISHED
	case domain.PropertyStatusSold:
		return pb.PropertyStatus_PROPERTY_STATUS_SOLD
	case domain.PropertyStatusDeleted:
		return pb.PropertyStatus_PROPERTY_STATUS_DELETED
	default:
		return pb.PropertyStatus_PROPERTY_STATUS_UNSPECIFIED
	}
}

func protoPropertyStatusToDomain(s pb.PropertyStatus) domain.PropertyStatus {
	switch s {
	case pb.PropertyStatus_PROPERTY_STATUS_NEW:
		return domain.PropertyStatusNew
	case pb.PropertyStatus_PROPERTY_STATUS_PUBLISHED:
		return domain.PropertyStatusPublished
	case pb.PropertyStatus_PROPERTY_STATUS_SOLD:
		return domain.PropertyStatusSold
	case pb.PropertyStatus_PROPERTY_STATUS_DELETED:
		return domain.PropertyStatusDeleted
	default:
		return domain.PropertyStatusUnspecified
	}
}

