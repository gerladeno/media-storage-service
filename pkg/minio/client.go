package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

const (
	getTimeoutSeconds       = 5
	getBucketTimeoutSeconds = 10
	uploadTimeoutSeconds    = 10
)

type Object struct {
	ID   string
	Size int64
	Tags map[string]string
}

type Client struct {
	log         *logrus.Entry
	minioClient *minio.Client
}

func NewClient(log *logrus.Logger, endpoint, accessKey, secretKey string) (*Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("err creating minio client")
	}
	return &Client{
		log:         log.WithField("module", "minio"),
		minioClient: minioClient,
	}, nil
}

func (c *Client) GetFile(ctx context.Context, bucketName, fileID string) (*minio.Object, error) {
	reqCtx, cancel := context.WithTimeout(ctx, getTimeoutSeconds*time.Second)
	defer cancel()

	obj, err := c.minioClient.GetObject(reqCtx, bucketName, fileID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file with id: %s from minio bucket %s. err: %w", fileID, bucketName, err)
	}
	return obj, nil
}

func (c *Client) GetBucketFiles(ctx context.Context, bucketName string) ([]*minio.Object, error) {
	reqCtx, cancel := context.WithTimeout(ctx, getBucketTimeoutSeconds*time.Second)
	defer cancel()

	var files []*minio.Object //nolint:prealloc
	for lobj := range c.minioClient.ListObjects(reqCtx, bucketName, minio.ListObjectsOptions{WithMetadata: true}) {
		if lobj.Err != nil {
			c.log.Warnf("failed to list object from minio bucket %s. err: %v", bucketName, lobj.Err)
			continue
		}
		object, err := c.minioClient.GetObject(ctx, bucketName, lobj.Key, minio.GetObjectOptions{})
		if err != nil {
			c.log.Warnf("failed to get object key=%s from minio bucket %s. err: %v", lobj.Key, bucketName, lobj.Err)
			continue
		}
		files = append(files, object)
	}
	return files, nil
}

func (c *Client) UploadFile(ctx context.Context, fileID, fileName, bucketName string, fileSize int64, reader io.Reader) error {
	reqCtx, cancel := context.WithTimeout(ctx, uploadTimeoutSeconds*time.Second)
	defer cancel()

	exists, errBucketExists := c.minioClient.BucketExists(ctx, bucketName)
	if errBucketExists != nil || !exists {
		c.log.Warnf("no bucket %s. creating new one...", bucketName)
		err := c.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create new bucket. err: %w", err)
		}
	}
	c.log.Debugf("put new object %s to bucket %s", fileName, bucketName)
	_, err := c.minioClient.PutObject(reqCtx, bucketName, fileID, reader, fileSize,
		minio.PutObjectOptions{
			UserMetadata: map[string]string{"Name": fileName},
			ContentType:  "application/octet-stream",
		})
	if err != nil {
		return fmt.Errorf("failed to upload file. err: %w", err)
	}
	return nil
}

func (c *Client) DeleteFile(ctx context.Context, noteUUID, fileName string) error {
	err := c.minioClient.RemoveObject(ctx, noteUUID, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file. err: %w", err)
	}
	return nil
}
