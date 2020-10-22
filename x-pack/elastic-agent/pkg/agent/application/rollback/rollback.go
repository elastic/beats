// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

// Rollback rollbacks to previous version which was functioning before upgrade.
func Rollback(prevHash, currentHash string) error {
	// TODO: finish me
	return nil
}

// Cleanup removes all artifacts and files related to a specified version.
func Cleanup(prevHash string) error {
	// TODO: finish me
	return nil

}

// InvokeWatcher invokes an agent instance using watcher argument for watching behavior of
// agent during upgrade period.
func InvokeWatcher() error {
	// TODO: finish me
	return nil
}
