package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v29/github"
)

type NewGitHubClientOption struct {
	// Required
	PrivateKey []byte
	// Required
	IntegrationID int

	// RepoOwner is required for installation API.
	RepoOwner string

	// Optional
	Client *http.Client
}

func NewGitHubClient(ctx context.Context, opt *NewGitHubClientOption) (*github.Client, error) {
	client := opt.Client
	if client == nil {
		client = http.DefaultClient
	}

	itr, err := githubAppTransport(ctx, client, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create gh transport: %v", err)
	}

	client.Transport = itr
	return github.NewClient(client), nil
}

func githubAppTransport(ctx context.Context, client *http.Client, opt *NewGitHubClientOption) (http.RoundTripper, error) {
	if opt.RepoOwner == "" {
		return ghinstallation.NewAppsTransport(getTransport(client), int64(opt.IntegrationID), opt.PrivateKey)
	}
	installationID, err := findInstallationID(ctx, opt)
	if err != nil {
		return nil, err
	}
	return ghinstallation.New(getTransport(client), int64(opt.IntegrationID), installationID, opt.PrivateKey)
}

func getTransport(client *http.Client) http.RoundTripper {
	if client.Transport != nil {
		return client.Transport
	}
	return http.DefaultTransport
}

func findInstallationID(ctx context.Context, opt *NewGitHubClientOption) (int64, error) {
	appCli, err := NewGitHubClient(ctx, &NewGitHubClientOption{
		PrivateKey:    opt.PrivateKey,
		IntegrationID: opt.IntegrationID,
		Client:        &http.Client{}, // Use different client to get installation.
		// Do no set RepoOwner.
	})
	if err != nil {
		return 0, err
	}
	inst, _, err := appCli.Apps.FindUserInstallation(ctx, opt.RepoOwner)
	if err != nil {
		return 0, err
	}
	return inst.GetID(), nil
}
