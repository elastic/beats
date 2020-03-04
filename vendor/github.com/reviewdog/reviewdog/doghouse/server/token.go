package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/reviewdog/reviewdog/doghouse/server/storage"
)

func GenerateRepositoryToken() string {
	return securerandom(8)
}

func securerandom(n int) string {
	b := make([]byte, n)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)
}

func GetOrGenerateRepoToken(ctx context.Context, s storage.GitHubRepositoryTokenStore,
	owner, repo string, repoID int64) (string, error) {
	found, token, err := s.Get(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	if found {
		return token.Token, nil
	}
	// If repo token not found, create a repo token.
	return RegenerateRepoToken(ctx, s, owner, repo, repoID)
}

func RegenerateRepoToken(ctx context.Context, s storage.GitHubRepositoryTokenStore,
	owner, repo string, repoID int64) (string, error) {
	newToken := &storage.GitHubRepositoryToken{
		RepositoryID:    repoID,
		RepositoryName:  repo,
		RepositoryOwner: owner,
		Token:           GenerateRepositoryToken(),
	}
	if err := s.Put(ctx, newToken); err != nil {
		return "", err
	}
	return newToken.Token, nil
}
