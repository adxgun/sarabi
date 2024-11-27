package storage

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"sarabi/types"
)

const (
	backupBucket = "backups"
)

type s3Storage struct {
	client *minio.Client
	region string
}

func NewS3Storage(cred types.StorageCredentials) (Storage, error) {
	mn, err := minio.New(cred.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cred.KeyId, cred.SecretKey, ""),
		Secure: false,
		Region: cred.Region,
	})
	if err != nil {
		return nil, err
	}
	return &s3Storage{
		region: cred.Region,
		client: mn,
	}, nil
}

func (s s3Storage) Save(ctx context.Context, location string, file types.File) error {
	if err := s.makeBucket(ctx); err != nil {
		return err
	}

	_, err := s.client.PutObject(ctx, backupBucket, location, bytes.NewReader(file.Content), file.Stat.Size, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s s3Storage) Get(ctx context.Context, location string) (io.Reader, error) {
	return s.client.GetObject(ctx, backupBucket, location, minio.GetObjectOptions{})
}

func (s s3Storage) makeBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, backupBucket)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return s.client.MakeBucket(ctx, backupBucket, minio.MakeBucketOptions{
		Region: s.region,
	})
}
