package bundler

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sarabi/types"
)

type (
	ArtifactStore interface {
		// Save writes the deployment binary to the host file system
		// the purpose of this is to enable support for deployment rollback.
		// there is a different provision for purging out the binaries once they're old(60 days by default)
		Save(ctx context.Context, artifact io.Reader, info *types.Deployment) error
		Copy(ctx context.Context, from, to *types.Deployment) error
	}
)

func NewArtifactStore() ArtifactStore {
	return &artifactStore{}
}

type artifactStore struct{}

func (a artifactStore) Save(ctx context.Context, artifact io.Reader, deployment *types.Deployment) error {
	dir := filepath.Dir(deployment.BinPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fi, err := os.Create(deployment.BinPath())
	if err != nil {
		return err
	}

	defer fi.Close()
	_, err = io.Copy(fi, artifact)
	if err != nil {
		return err
	}

	return nil
}

func (a artifactStore) Copy(ctx context.Context, from, to *types.Deployment) error {
	src, err := os.Open(from.BinPath())
	if err != nil {
		return err
	}

	return a.Save(ctx, src, to)
}
