package storage

import (
	"context"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/datastore"
)

// datastoreClient constructs a *datastore.Client using runtime introspection
// to target the current project's datastore.
func datastoreClient(ctx context.Context) (*datastore.Client, error) {
	id, err := metadata.ProjectID()
	if err != nil {
		return nil, err
	}
	return datastore.NewClient(ctx, id)
}
