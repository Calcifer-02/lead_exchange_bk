package filegrpc

import (
	"context"
	"lead_exchange/internal/domain"
	desc "lead_exchange/pkg"
)

func (s *fileServer) UploadFile(ctx context.Context, in *desc.UploadFileRequest) (*desc.UploadFileResponse, error) {
	err := in.ValidateAll()
	if err != nil {
		return nil, err
	}

	file := domain.FileDataType{
		FileName: in.FileName,
		Data:     in.File,
	}

	url, err := s.minioClient.CreateOne(file)
	if err != nil {
		return nil, err
	}

	return &desc.UploadFileResponse{
		Url: url,
	}, nil
}
