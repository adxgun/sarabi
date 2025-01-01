package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	types "sarabi/internal/types"
)

const (
	backupBucket = "backups"
)

type objectStorage struct {
	client *minio.Client
	region string
}

func NewObjectStorage(cred types.StorageCredentials) (Storage, error) {
	mn, err := minio.New(cred.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cred.AccessKeyID, cred.SecretKey, ""),
		Secure: false,
		Region: cred.Region,
	})
	if err != nil {
		return nil, err
	}
	return &objectStorage{
		region: cred.Region,
		client: mn,
	}, nil
}

func (s objectStorage) Save(ctx context.Context, location string, file types.File) error {
	if err := s.makeBucket(ctx); err != nil {
		return err
	}

	// TODO: implement chunk writer
	_, err := s.client.PutObject(ctx, backupBucket, location, file.Content, file.Stat.Size, minio.PutObjectOptions{
		ContentType: file.GetContentType(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s objectStorage) Get(ctx context.Context, location string) (*types.File, error) {
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

func (s objectStorage) makeBucket(ctx context.Context) error {
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

func (s objectStorage) Ping(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx)
	if err != nil {
		return err
	}
	return nil
}
