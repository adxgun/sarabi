package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	// TODO: implement chunk writer
	_, err := s.client.PutObject(ctx, backupBucket, location, file.Content, file.Stat.Size, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s s3Storage) Get(ctx context.Context, location string) (*types.File, error) {
	r, err := s.client.GetObject(ctx, backupBucket, location, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	stat, err := r.Stat()
	if err != nil {
		return nil, err
	}

	return &types.File{
		Content: r,
		Stat:    types.FileStat{Size: stat.Size, Name: stat.Key},
	}, nil
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
