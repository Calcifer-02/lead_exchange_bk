package filegrpc

import (
	minio "lead_exchange/internal/lib/minio/core"
	pb "lead_exchange/pkg"

	"google.golang.org/grpc"
)

// fileServer реализует gRPC FileServiceServer.
type fileServer struct {
	pb.UnimplementedFileServiceServer

	minioClient minio.Client
}

// RegisterFileServerGRPC регистрирует FileServiceServer в gRPC сервере.
func RegisterFileServerGRPC(server *grpc.Server, minioClient minio.Client) {
	pb.RegisterFileServiceServer(server, &fileServer{
		minioClient: minioClient,
	})
}
