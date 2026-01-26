// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package views

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var registry []ViewSpec

// RegisterViewSpec registers a view spec.
// This is called from each generated view's init() function.
func RegisterViewSpec(spec ViewSpec) {
	registry = append(registry, spec)
}

// ViewSpec contains metadata and references for a generated view.
type ViewSpec struct {
	Name           string
	Description    string
	Platforms      []string
	RequiredTables []string
	View           func() *hooks.View
	HooksFunc      func(*hooks.HookManager)
}

// RegisterViews registers all views in the registry as hooks.
// Each view is registered as a hook that creates the view in osquery when executed.
func RegisterViews(hookManager *hooks.HookManager) {
	for _, spec := range registry {
		view := spec.View()
		
		// Create a hook that will create the view
		hook := hooks.NewHook(
			"view_"+spec.Name,
			func(socket *string, log *logger.Logger, hookData any) error {
				v := hookData.(*hooks.View)
				return v.Create(socket, log)
			},
			func(socket *string, log *logger.Logger, hookData any) error {
				v := hookData.(*hooks.View)
				return v.Delete(socket, log)
			},
			view,
		)
		hookManager.Register(hook)
		
		// Register additional hooks if provided
		if spec.HooksFunc != nil {
			spec.HooksFunc(hookManager)
		}
	}
}

