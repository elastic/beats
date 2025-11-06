// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"sync"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// HookFunc represents a function that executes a hook, any
// function that implements this interface can be used as a hook
type HookFunc func(socket *string, log *logger.Logger, hookData any) error

// Hook is a struct that represents a hook function
type Hook struct {
	hookName     string
	hookFunc     HookFunc
	shutdownFunc HookFunc
	hookData     any
}

// Name returns the name of the hook
func (h *Hook) Name() string {
	return h.hookName
}

// Execute executes the hook function
func (h *Hook) Execute(socket *string, log *logger.Logger) error {
	if h.hookFunc == nil {
		return nil
	}
	log.Infof("executing hook, name: %s", h.Name())
	return h.hookFunc(socket, log, h.hookData)
}

func (h *Hook) Shutdown(socket *string, log *logger.Logger) error {
	if h.shutdownFunc == nil {
		return nil
	}
	log.Infof("executing shutdown hook, name: %s", h.Name())
	return h.shutdownFunc(socket, log, h.hookData)
}

// NewHook creates a new hook
func NewHook(hookName string, hookFunc HookFunc, shutdownFunc HookFunc, hookData any) *Hook {
	return &Hook{hookName: hookName, hookFunc: hookFunc, shutdownFunc: shutdownFunc, hookData: hookData}
}

// HookManager is a struct that contains all hooks of a given type
type HookManager struct {
	hooks []*Hook
}

// Register registers a new hook
func (hm *HookManager) Register(hook *Hook) {
	hm.hooks = append(hm.hooks, hook)
}

// Execute executes all hooks of a given type concurrently
func (hm *HookManager) Execute(socket *string, log *logger.Logger) {
	wg := &sync.WaitGroup{}
	wg.Add(len(hm.hooks))
	for _, hook := range hm.hooks {
		go func(hook *Hook) {
			defer wg.Done()
			err := hook.Execute(socket, log)
			if err != nil {
				log.Errorf("error executing hook, name: %s, error: %v", hook.Name(), err)
			}
		}(hook)
	}
	wg.Wait()
}

func (hm *HookManager) Shutdown(socket *string, log *logger.Logger) {
	wg := &sync.WaitGroup{}
	wg.Add(len(hm.hooks))
	for _, hook := range hm.hooks {
		go func(hook *Hook) {
			defer wg.Done()
			err := hook.Shutdown(socket, log)
			if err != nil {
				log.Errorf("error executing hook, name: %s, error: %v", hook.Name(), err)
			}
		}(hook)
	}
	wg.Wait()
}

func NewHookManager() *HookManager {
	return &HookManager{hooks: make([]*Hook, 0)}
}
