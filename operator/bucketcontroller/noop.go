package bucketcontroller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// NoopClient is a client that does nothing.
type NoopClient struct{}

// Observe implements managed.ExternalClient.
// It always returns an observation where the resource doesn't exist and is outdated, together with a nil error.
func (n *NoopClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	return managed.ExternalObservation{}, nil
}

// Create implement managed.ExternalClient.
// It returns nil.
func (n *NoopClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, nil
}

// Update implement managed.ExternalClient.
// It returns nil.
func (n *NoopClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

// Delete implement managed.ExternalClient.
// It returns nil.
func (n *NoopClient) Delete(ctx context.Context, mg resource.Managed) error {
	return nil
}
