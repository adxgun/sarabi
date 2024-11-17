package components

import (
	"context"
	"github.com/google/uuid"
	"sarabi/types"
)

type BuilderResult struct {
	ID             string
	Name           string
	PreviousActive []*types.Deployment
}

type Builder interface {
	Name() string
	Run(ctx context.Context, deploymentID uuid.UUID) (*BuilderResult, error)
	Cleanup(ctx context.Context, result *BuilderResult) error
}
