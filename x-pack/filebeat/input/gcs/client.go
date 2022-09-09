// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"errors"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchStorageClient(ctx context.Context, cfg config, log *logp.Logger) (*storage.Client, error) {
	if cfg.Auth.CredentialsJSON != nil {
		return storage.NewClient(ctx, option.WithCredentialsJSON([]byte(cfg.Auth.CredentialsJSON.AccountKey)))
	} else if cfg.Auth.CredentialsFile != nil {
		return storage.NewClient(ctx, option.WithCredentialsFile(cfg.Auth.CredentialsFile.Path))
	}
	return nil, errors.New("no valid auth specified")
}
