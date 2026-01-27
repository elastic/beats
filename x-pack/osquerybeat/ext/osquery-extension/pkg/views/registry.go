// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package views

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/views/generated"
)

// RegisterViews registers all generated views with the hook manager.
// This is the stable entry point that wraps the generated registry.
func RegisterViews(hookManager *hooks.HookManager, log *logger.Logger) {
	generated.RegisterViews(hookManager, log)
}
