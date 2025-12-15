package filegrpc

import (
	"context"
	"lead_exchange/internal/domain"
	desc "lead_exchange/pkg"
)

func (s *fileServer) UploadFiles(ctx context.Context, in *desc.UploadFilesRequest) (*desc.UploadFilesResponse, error) {
	err := in.ValidateAll()
	if err != nil {
		return nil, err
	}

	filesMap := make(map[string]domain.FileDataType)
	for _, f := range in.Files {
		filesMap[f.FileName] = domain.FileDataType{
			FileName: f.FileName,
			Data:     f.File,
		}
	}

	urls, err := s.minioClient.CreateMany(filesMap)
	if err != nil {
		return nil, err
	}

	return &desc.UploadFilesResponse{
		Urls: urls,
	}, nil
}
