package storage

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
)

// GitHubRepositoryToken represents token data for authenticating reviewdog CLI
// to the target repository.
type GitHubRepositoryToken struct {
	Token string
	// https://github/<RepositoryOwner>/<RepositoryName>.
	RepositoryOwner string
	RepositoryName  string
	RepositoryID    int64
}

// GitHubRepositoryTokenStore represents GitHubRepositoryToken storage interface.
type GitHubRepositoryTokenStore interface {
	// Put upserts GitHubRepositoryToken entity.
	Put(ctx context.Context, token *GitHubRepositoryToken) error
	// Get get GitHubRepositoryToken entity by owner and repo name.
	// - If the entity is found, return token with ok is true.
	// - If the entity is not found, ok is false.
	// - If error occurs, it returns err.
	Get(ctx context.Context, owner, repo string) (ok bool, token *GitHubRepositoryToken, err error)
}

// GitHubRepoTokenDatastore is store of GitHubRepositoryToken by Datastore of
// Google Appengine.
type GitHubRepoTokenDatastore struct{}

func (g *GitHubRepoTokenDatastore) newKey(owner, repo string) *datastore.Key {
	kind := "GitHubRepositoryToken"
	return datastore.NameKey(kind, fmt.Sprintf("%s/%s", owner, repo), nil)
}

// Put upserts GitHubRepositoryToken.
func (g *GitHubRepoTokenDatastore) Put(ctx context.Context, token *GitHubRepositoryToken) error {
	key := g.newKey(token.RepositoryOwner, token.RepositoryName)
	d, err := datastoreClient(ctx)
	if err != nil {
		return err
	}
	_, err = d.Put(ctx, key, token)
	return err
}

func (g *GitHubRepoTokenDatastore) Get(ctx context.Context, owner, repo string) (ok bool, token *GitHubRepositoryToken, err error) {
	key := g.newKey(owner, repo)
	token = new(GitHubRepositoryToken)
	d, err := datastoreClient(ctx)
	if err != nil {
		return false, nil, err
	}
	if err := d.Get(ctx, key, token); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, token, nil
}
