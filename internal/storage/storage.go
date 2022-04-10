package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gerladeno/media-storage-service/internal/apperror"
	"github.com/gerladeno/media-storage-service/pkg/minio"
	"github.com/sirupsen/logrus"
)

type Storage interface {
	GetFile(ctx context.Context, bucketName, fileName string) (*File, error)
	GetFilesByNoteUUID(ctx context.Context, uuid string) ([]*File, error)
	CreateFile(ctx context.Context, noteUUID string, file *File) error
	DeleteFile(ctx context.Context, noteUUID, fileName string) error
}

type File struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Bytes []byte `json:"bytes"`
}

type CreateFileDTO struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Reader io.Reader
}

func NewFile(dto CreateFileDTO) (*File, error) {
	bytes, err := ioutil.ReadAll(dto.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create file model. err: %w", err)
	}
	sum := sha256.Sum256(append([]byte(dto.Name), bytes...))
	name := base64.URLEncoding.EncodeToString(sum[:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate file id. err: %w", err)
	}

	return &File{
		ID:    name,
		Name:  dto.Name,
		Size:  dto.Size,
		Bytes: bytes,
	}, nil
}

type minioStorage struct {
	log    *logrus.Entry
	client *minio.Client
}

func New(log *logrus.Logger, endpoint, accessKey, secretKey string) (Storage, error) {
	client, err := minio.NewClient(log, endpoint, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client. err: %w", err)
	}
	return &minioStorage{
		log:    log.WithField("module", "storage"),
		client: client,
	}, nil
}

func (m *minioStorage) GetFile(ctx context.Context, bucketName, fileID string) (*File, error) {
	obj, err := m.client.GetFile(ctx, bucketName, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file. err: %w", err)
	}
	defer obj.Close()
	objectInfo, err := obj.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file. err: %w", err)
	}
	buffer := make([]byte, objectInfo.Size)
	_, err = obj.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to get objects. err: %w", err)
	}
	f := File{
		ID:    objectInfo.Key,
		Name:  objectInfo.UserMetadata["Name"],
		Size:  objectInfo.Size,
		Bytes: buffer,
	}
	return &f, nil
}

func (m *minioStorage) GetFilesByNoteUUID(ctx context.Context, noteUUID string) ([]*File, error) {
	objects, err := m.client.GetBucketFiles(ctx, noteUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get objects. err: %w", err)
	}
	if len(objects) == 0 {
		return nil, apperror.ErrNotFound
	}

	files := make([]*File, 0, len(objects))
	for _, obj := range objects {
		stat, err := obj.Stat()
		if err != nil {
			m.log.Warnf("failed to get objects. err: %v", err)
			continue
		}
		buffer := make([]byte, stat.Size)
		_, err = obj.Read(buffer)
		if err != nil && err != io.EOF {
			m.log.Warnf("failed to get objects. err: %v", err)
			continue
		}
		f := File{
			ID:    stat.Key,
			Name:  stat.UserMetadata["Name"],
			Size:  stat.Size,
			Bytes: buffer,
		}
		files = append(files, &f)
		_ = obj.Close()
	}

	return files, nil
}

func (m *minioStorage) CreateFile(ctx context.Context, noteUUID string, file *File) error {
	err := m.client.UploadFile(ctx, file.ID, file.Name, noteUUID, file.Size, bytes.NewBuffer(file.Bytes))
	if err != nil {
		return err
	}
	return nil
}

func (m *minioStorage) DeleteFile(ctx context.Context, noteUUID, fileID string) error {
	err := m.client.DeleteFile(ctx, noteUUID, fileID)
	if err != nil {
		return err
	}
	return nil
}
