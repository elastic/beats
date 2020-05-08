package storage

import (
	"context"

	"cloud.google.com/go/datastore"
)

// GitHubInstallation represents GitHub Apps Installation data.
// Installation is per org or user account, not repository.
type GitHubInstallation struct {
	InstallationID int64  // https://github.com/settings/installations/<InstallationID>
	AccountName    string // https://github/<AccountName>. Org or user account.
	AccountID      int64  // GitHub ID of <AccountName>.
}

// GitHubInstallationStore represents GitHubInstallation storage interface.
type GitHubInstallationStore interface {
	// Put upserts GitHub InstallationID entity. If InstallationID is not
	// updated, the whole entity won't be saved.
	Put(ctx context.Context, inst *GitHubInstallation) error
	// Get get GitHubInstallation entity by account name.
	// - If the entity is found, return inst with ok is true.
	// - If the entity is not found, ok is false.
	// - If error occurs, it returns err.
	Get(ctx context.Context, accountName string) (ok bool, inst *GitHubInstallation, err error)
}

// GitHubInstallationDatastore is store of GitHubInstallation by Datastore of
// Google Appengine.
type GitHubInstallationDatastore struct{}

func (g *GitHubInstallationDatastore) newKey(accountName string) *datastore.Key {
	const kind = "GitHubInstallation"
	return datastore.NameKey(kind, accountName, nil)
}

// Put save GitHubInstallation. It reduces datastore write call as much as possible.
func (g *GitHubInstallationDatastore) Put(ctx context.Context, inst *GitHubInstallation) error {
	d, err := datastoreClient(ctx)
	if err != nil {
		return err
	}
	_, err = d.RunInTransaction(ctx, func(t *datastore.Transaction) error {
		var foundInst GitHubInstallation
		var ok bool
		err := t.Get(g.newKey(inst.AccountName), &foundInst)
		if err != datastore.ErrNoSuchEntity {
			ok = true
		}
		if err != nil {
			return err
		}
		// Insert if not found or installation ID is different.
		if !ok || foundInst.InstallationID != inst.InstallationID {
			_, err = t.Put(g.newKey(inst.AccountName), inst)
			return err
		}
		return nil // Do nothing.
	})
	return err
}

func (g *GitHubInstallationDatastore) Get(ctx context.Context, accountName string) (ok bool, inst *GitHubInstallation, err error) {
	key := g.newKey(accountName)
	inst = new(GitHubInstallation)
	d, err := datastoreClient(ctx)
	if err != nil {
		return false, nil, err
	}
	if err := d.Get(ctx, key, inst); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, inst, nil
}
