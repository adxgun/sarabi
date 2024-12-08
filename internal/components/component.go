package components

import (
	"context"
	"github.com/google/uuid"
	"sarabi/internal/types"
)

type BuilderResult struct {
	ID             string
	Name           string
	PreviousActive []*types.Deployment
}

// Builder is a component builder. Component here means a part that make up a fullstack application(backend, frontend, database, proxy)
// this interface is the representation of the process of managing each component
type Builder interface {
	// Name returns the name of the specific component - just for identification
	Name() string

	// Run is responsible for starting the concrete component. implementation defers depending on the actual component.
	// e.g for backend - it builds docker image from uploaded build, start the server container(s) and update proxy config to make the app accessible
	Run(ctx context.Context, deploymentID uuid.UUID) (*BuilderResult, error)

	// Cleanup is responsible for cleaning up the stale/excess resources after a component is created.
	// implementation depends on the actual component.
	// e.g for backend, remove old instances of the backend containers and update their status to STOPPED
	Cleanup(ctx context.Context, result *BuilderResult) error
}
