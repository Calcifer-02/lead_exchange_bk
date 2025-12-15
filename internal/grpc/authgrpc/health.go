package authgrpc

import (
	"context"
	pb "lead_exchange/pkg"

	"google.golang.org/protobuf/types/known/emptypb"
)

// HealthCheck — проверка доступности сервера.
func (s *authServer) HealthCheck(ctx context.Context, _ *emptypb.Empty) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: "ok"}, nil
}
