package storage

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Service struct {
	log     *logrus.Entry
	storage Storage
}

func NewService(log *logrus.Logger, noteStorage Storage) (*Service, error) {
	return &Service{
		storage: noteStorage,
		log:     log.WithField("module", "service"),
	}, nil
}

func (s *Service) GetFile(ctx context.Context, noteUUID, fileID string) (f *File, err error) {
	f, err = s.storage.GetFile(ctx, noteUUID, fileID)
	if err != nil {
		return f, err
	}
	return f, nil
}

func (s *Service) GetFilesByNoteUUID(ctx context.Context, noteUUID string) ([]*File, error) {
	files, err := s.storage.GetFilesByNoteUUID(ctx, noteUUID)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *Service) Create(ctx context.Context, noteUUID string, dto CreateFileDTO) error {
	file, err := NewFile(dto)
	if err != nil {
		return err
	}
	err = s.storage.CreateFile(ctx, noteUUID, file)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, noteUUID, fileName string) error {
	err := s.storage.DeleteFile(ctx, noteUUID, fileName)
	if err != nil {
		return err
	}
	return nil
}
