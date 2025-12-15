package usergrpc

import (
	"lead_exchange/internal/domain"
	pb "lead_exchange/pkg"
)

// userDomainToProto — преобразует доменную сущность пользователя в protobuf-модель.
func userDomainToProto(u domain.User) *pb.UserProfile {
	return &pb.UserProfile{
		Id:         u.ID.String(),
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Phone:      u.Phone,
		AgencyName: u.AgencyName,
		AvatarUrl:  u.AvatarURL,
		Role:       userTypeDomainToProto(u.Role),
		Status:     userStatusDomainToProto(u.Status),
	}
}

func userTypeDomainToProto(userType domain.UserRole) pb.UserRole {
	switch userType {
	case domain.UserRoleUser:
		return pb.UserRole_USER_ROLE_USER
	case domain.UserRoleAdmin:
		return pb.UserRole_USER_ROLE_ADMIN
	default:
		return pb.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func userRoleProtoToDomain(role pb.UserRole) domain.UserRole {
	switch role {
	case pb.UserRole_USER_ROLE_USER:
		return domain.UserRoleUser
	case pb.UserRole_USER_ROLE_ADMIN:
		return domain.UserRoleAdmin
	default:
		return domain.UserRoleUnspecified
	}
}

func userStatusDomainToProto(status domain.UserStatus) pb.UserStatus {
	switch status {
	case domain.UserStatusActive:
		return pb.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusBanned:
		return pb.UserStatus_USER_STATUS_BANNED
	case domain.UserStatusSuspended:
		return pb.UserStatus_USER_STATUS_SUSPENDED
	default:
		return pb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func userStatusProtoToDomain(status pb.UserStatus) domain.UserStatus {
	switch status {
	case pb.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive
	case pb.UserStatus_USER_STATUS_BANNED:
		return domain.UserStatusBanned
	case pb.UserStatus_USER_STATUS_SUSPENDED:
		return domain.UserStatusSuspended
	default:
		return domain.UserStatusUnspecified
	}
}
