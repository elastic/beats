// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package bundle implements bundle loading.
package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
	"github.com/open-policy-agent/opa/download"
	bundleUtils "github.com/open-policy-agent/opa/internal/bundle"
	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/storage"
)

// Loader defines the interface that the bundle plugin uses to control bundle
// loading via HTTP, disk, etc.
type Loader interface {
	Start(context.Context)
	Stop(context.Context)
	Trigger(context.Context) error
	SetCache(string)
	ClearCache()
}

// Plugin implements bundle activation.
type Plugin struct {
	config            Config
	manager           *plugins.Manager                         // plugin manager for storage and service clients
	status            map[string]*Status                       // current status for each bundle
	etags             map[string]string                        // etag on last successful activation
	listeners         map[interface{}]func(Status)             // listeners to send status updates to
	bulkListeners     map[interface{}]func(map[string]*Status) // listeners to send aggregated status updates to
	downloaders       map[string]Loader
	logger            logging.Logger
	mtx               sync.Mutex
	cfgMtx            sync.Mutex
	ready             bool
	bundlePersistPath string
	stopped           bool
}

// New returns a new Plugin with the given config.
func New(parsedConfig *Config, manager *plugins.Manager) *Plugin {
	initialStatus := map[string]*Status{}
	for name := range parsedConfig.Bundles {
		initialStatus[name] = &Status{
			Name: name,
		}
	}

	p := &Plugin{
		manager:     manager,
		config:      *parsedConfig,
		status:      initialStatus,
		downloaders: make(map[string]Loader),
		etags:       make(map[string]string),
		ready:       false,
		logger:      manager.Logger(),
	}

	manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateNotReady})
	return p
}

// Name identifies the plugin on manager.
const Name = "bundle"

// Lookup returns the bundle plugin registered with the manager.
func Lookup(manager *plugins.Manager) *Plugin {
	if p := manager.Plugin(Name); p != nil {
		return p.(*Plugin)
	}
	return nil
}

// Start runs the plugin. The plugin will periodically try to download bundles
// from the configured service. When a new bundle is downloaded, the data and
// policies are extracted and inserted into storage.
func (p *Plugin) Start(ctx context.Context) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	var err error

	p.bundlePersistPath, err = p.getBundlePersistPath()
	if err != nil {
		return err
	}

	err = p.loadAndActivateBundlesFromDisk(ctx)
	if err != nil {
		return err
	}

	p.initDownloaders()
	for name, dl := range p.downloaders {
		p.log(name).Info("Starting bundle loader.")
		dl.Start(ctx)
	}
	return nil
}

// Stop stops the plugin.
func (p *Plugin) Stop(ctx context.Context) {
	p.mtx.Lock()
	stopDownloaders := map[string]Loader{}
	for name, dl := range p.downloaders {
		stopDownloaders[name] = dl
	}
	p.downloaders = nil
	p.stopped = true
	p.mtx.Unlock()

	for name, dl := range stopDownloaders {
		p.log(name).Info("Stopping bundle loader.")
		dl.Stop(ctx)
	}
}

// Reconfigure notifies the plugin that it's configuration has changed.
// Any bundle configs that have changed or been added/removed will take
// affect.
func (p *Plugin) Reconfigure(ctx context.Context, config interface{}) {
	// Reconfiguring should not occur in parallel, lock to ensure
	// nothing swaps underneath us with the current p.config and the updated one.
	// Use p.cfgMtx instead of p.mtx so as to not block any bundle downloads/activations
	// that are in progress. We upgrade to p.mtx locking after stopping downloaders.
	p.cfgMtx.Lock()
	defer p.cfgMtx.Unlock()

	// Look for any bundles that have had their config changed, are new, or have been removed
	newConfig := config.(*Config)
	newBundles, updatedBundles, deletedBundles := p.configDelta(newConfig)
	p.config = *newConfig

	if len(updatedBundles) == 0 && len(newBundles) == 0 && len(deletedBundles) == 0 {
		// no relevant config changes
		return
	}

	// Stop the downloaders outside p.mtx to allow them to finish handling any in-progress requests.
	for name, dl := range p.downloaders {
		_, updated := updatedBundles[name]
		_, deleted := deletedBundles[name]
		if updated || deleted {
			dl.Stop(ctx)
		}
	}

	// Only lock p.mtx once we start changing the internal maps
	// and downloader configs.
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// Cleanup existing downloaders that are deleted
	for name := range p.downloaders {
		if _, deleted := deletedBundles[name]; deleted {
			p.log(name).Info("Bundle loader configuration removed. Stopping bundle loader.")
			delete(p.downloaders, name)
			delete(p.status, name)
			delete(p.etags, name)
		}
	}

	// Deactivate the bundles that were removed
	params := storage.WriteParams
	params.Context = storage.NewContext()
	err := storage.Txn(ctx, p.manager.Store, params, func(txn storage.Transaction) error {
		opts := &bundle.DeactivateOpts{
			Ctx:         ctx,
			Store:       p.manager.Store,
			Txn:         txn,
			BundleNames: deletedBundles,
		}
		err := bundle.Deactivate(opts)
		if err != nil {
			p.manager.Logger().Error(fmt.Sprint(deletedBundles), "Failed to deactivate bundles: %s", err)
			return err
		}
		return nil
	})
	if err != nil {
		// TODO(patrick-east): This probably shouldn't panic.. But OPA shouldn't
		// continue in a potentially inconsistent state.
		panic(errors.New("Unable deactivate bundle: " + err.Error()))
	}

	readyNow := p.ready

	for name, source := range p.config.Bundles {
		_, updated := updatedBundles[name]
		_, isNew := newBundles[name]

		if isNew || updated {
			if isNew {
				p.status[name] = &Status{Name: name}
				p.log(name).Info("New bundle loader configuration added. Starting bundle loader.")
			} else {
				p.log(name).Info("Bundle loader configuration changed. Restarting bundle loader.")
			}
			p.downloaders[name] = p.newDownloader(name, source)
			p.downloaders[name].Start(ctx)
			readyNow = false
		}
	}

	if !readyNow {
		p.ready = false
		p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateNotReady})
	}

}

// Loaders returns the map of bundle loaders configured on this plugin.
func (p *Plugin) Loaders() map[string]Loader {
	return p.downloaders
}

// Trigger triggers a bundle download on all configured bundles.
func (p *Plugin) Trigger(ctx context.Context) error {
	p.mtx.Lock()
	downloaders := map[string]Loader{}
	for name, dl := range p.downloaders {
		downloaders[name] = dl
	}
	p.mtx.Unlock()

	for _, d := range downloaders {
		// plugin callback will log the trigger error and include it in the bundle status
		_ = d.Trigger(ctx)
	}
	return nil
}

// Register a listener to receive status updates. The name must be comparable.
// The listener will receive a status update for each bundle configured, they are
// not going to be aggregated. For all status updates use `RegisterBulkListener`.
func (p *Plugin) Register(name interface{}, listener func(Status)) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if p.listeners == nil {
		p.listeners = map[interface{}]func(Status){}
	}

	p.listeners[name] = listener
}

// Unregister a listener to stop receiving status updates.
func (p *Plugin) Unregister(name interface{}) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	delete(p.listeners, name)
}

// RegisterBulkListener registers a listener to receive bulk (aggregated) status updates. The name must be comparable.
func (p *Plugin) RegisterBulkListener(name interface{}, listener func(map[string]*Status)) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if p.bulkListeners == nil {
		p.bulkListeners = map[interface{}]func(map[string]*Status){}
	}

	p.bulkListeners[name] = listener
}

// UnregisterBulkListener unregisters a listener to stop receiving aggregated status updates.
func (p *Plugin) UnregisterBulkListener(name interface{}) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	delete(p.bulkListeners, name)
}

// Config returns the plugins current configuration
func (p *Plugin) Config() *Config {
	return &p.config
}

func (p *Plugin) initDownloaders() {
	// Initialize a downloader for each bundle configured.
	for name, source := range p.config.Bundles {
		p.downloaders[name] = p.newDownloader(name, source)
	}
}

func (p *Plugin) loadAndActivateBundlesFromDisk(ctx context.Context) error {
	for name, src := range p.config.Bundles {
		if p.persistBundle(name) {
			b, err := loadBundleFromDisk(p.bundlePersistPath, name, src)
			if err != nil {
				p.log(name).Error("Failed to load bundle from disk: %v", err)
				return err
			}

			if b == nil {
				return nil
			}

			p.status[name].Metrics = metrics.New()

			err = p.activate(ctx, name, b)
			if err != nil {
				p.log(name).Error("Bundle activation failed: %v", err)
				return err
			}

			p.status[name].SetError(nil)
			p.status[name].SetActivateSuccess(b.Manifest.Revision)

			p.checkPluginReadiness()

			p.log(name).Debug("Bundle loaded from disk and activated successfully.")
		}
	}
	return nil
}

func (p *Plugin) newDownloader(name string, source *Source) Loader {

	if u, err := url.Parse(source.Resource); err == nil {
		switch u.Scheme {
		case "file":
			return &fileLoader{
				name:           name,
				path:           u.Path,
				bvc:            source.Signing,
				sizeLimitBytes: source.SizeLimitBytes,
				f:              p.oneShot,
			}
		}
	}

	conf := source.Config
	client := p.manager.Client(source.Service)
	path := source.Resource
	callback := func(ctx context.Context, u download.Update) {
		// wrap the callback to include the name of the bundle that was updated
		p.oneShot(ctx, name, u)
	}
	return download.New(conf, client, path).
		WithCallback(callback).
		WithBundleVerificationConfig(source.Signing).
		WithSizeLimitBytes(source.SizeLimitBytes).
		WithBundlePersistence(p.persistBundle(name))
}

func (p *Plugin) oneShot(ctx context.Context, name string, u download.Update) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.process(ctx, name, u)

	for _, listener := range p.listeners {
		listener(*p.status[name])
	}

	for _, listener := range p.bulkListeners {
		// Send a copy of the full status map to the bulk listeners.
		// They shouldn't have access to the original underlying
		// map, primarily for thread safety issues with modifications
		// made to it.
		statusCpy := map[string]*Status{}
		for k, v := range p.status {
			v := *v
			statusCpy[k] = &v
		}
		listener(statusCpy)
	}
}

func (p *Plugin) process(ctx context.Context, name string, u download.Update) {

	if u.Metrics != nil {
		p.status[name].Metrics = u.Metrics
	} else {
		p.status[name].Metrics = metrics.New()
	}

	p.status[name].SetRequest()

	if u.Error != nil {
		p.log(name).Error("Bundle load failed: %v", u.Error)
		p.status[name].SetError(u.Error)
		if !p.stopped {
			etag := p.etags[name]
			p.downloaders[name].SetCache(etag)
		}
		return
	}

	p.status[name].LastSuccessfulRequest = p.status[name].LastRequest

	if u.Bundle != nil {
		p.status[name].LastSuccessfulDownload = p.status[name].LastSuccessfulRequest

		p.status[name].Metrics.Timer(metrics.RegoLoadBundles).Start()
		defer p.status[name].Metrics.Timer(metrics.RegoLoadBundles).Stop()

		if err := p.activate(ctx, name, u.Bundle); err != nil {
			p.log(name).Error("Bundle activation failed: %v", err)
			p.status[name].SetError(err)
			if !p.stopped {
				etag := p.etags[name]
				p.downloaders[name].SetCache(etag)
			}
			return
		}

		if p.persistBundle(name) {
			p.log(name).Debug("Persisting bundle to disk in progress.")

			err := p.saveBundleToDisk(name, u.Raw)
			if err != nil {
				p.log(name).Error("Persisting bundle to disk failed: %v", err)
				p.status[name].SetError(err)
				if !p.stopped {
					etag := p.etags[name]
					p.downloaders[name].SetCache(etag)
				}
				return
			}
			p.log(name).Debug("Bundle persisted to disk successfully at path %v.", filepath.Join(p.bundlePersistPath, name))
		}

		p.status[name].SetError(nil)
		p.status[name].SetActivateSuccess(u.Bundle.Manifest.Revision)

		if u.ETag != "" {
			p.log(name).Info("Bundle loaded and activated successfully. Etag updated to %v.", u.ETag)
		} else {
			p.log(name).Info("Bundle loaded and activated successfully.")
		}
		p.etags[name] = u.ETag

		// If the plugin wasn't ready yet then check if we are now after activating this bundle.
		p.checkPluginReadiness()
		return
	}

	if etag, ok := p.etags[name]; ok && u.ETag == etag {
		p.log(name).Debug("Bundle load skipped, server replied with not modified.")
		p.status[name].SetError(nil)
		return
	}
}

func (p *Plugin) checkPluginReadiness() {
	if !p.ready {
		readyNow := true // optimistically
		for _, status := range p.status {
			if len(status.Errors) > 0 || (status.LastSuccessfulActivation == time.Time{}) {
				readyNow = false // Not ready yet, check again on next bundle activation.
				break
			}
		}

		if readyNow {
			p.ready = true
			p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateOK})
		}
	}
}

func (p *Plugin) activate(ctx context.Context, name string, b *bundle.Bundle) error {
	p.log(name).Debug("Bundle activation in progress. Opening storage transaction.")

	params := storage.WriteParams
	params.Context = storage.NewContext()

	err := storage.Txn(ctx, p.manager.Store, params, func(txn storage.Transaction) error {
		p.log(name).Debug("Opened storage transaction (%v).", txn.ID())
		defer p.log(name).Debug("Closing storage transaction (%v).", txn.ID())

		// Compile the bundle modules with a new compiler and set it on the
		// transaction params for use by onCommit hooks.
		compiler := ast.NewCompiler().
			WithPathConflictsCheck(storage.NonEmpty(ctx, p.manager.Store, txn)).
			WithEnablePrintStatements(p.manager.EnablePrintStatements())

		var activateErr error

		opts := &bundle.ActivateOpts{
			Ctx:      ctx,
			Store:    p.manager.Store,
			Txn:      txn,
			TxnCtx:   params.Context,
			Compiler: compiler,
			Metrics:  p.status[name].Metrics,
			Bundles:  map[string]*bundle.Bundle{name: b},
		}

		if p.config.IsMultiBundle() {
			activateErr = bundle.Activate(opts)
		} else {
			activateErr = bundle.ActivateLegacy(opts)
		}

		plugins.SetCompilerOnContext(params.Context, compiler)

		resolvers, err := bundleUtils.LoadWasmResolversFromStore(ctx, p.manager.Store, txn, nil)
		if err != nil {
			return err
		}

		plugins.SetWasmResolversOnContext(params.Context, resolvers)

		return activateErr
	})

	return err
}

func (p *Plugin) persistBundle(name string) bool {
	bundleSrc := p.config.Bundles[name]

	if bundleSrc == nil {
		return false
	}
	return bundleSrc.Persist
}

// configDelta will return a map of new bundle sources, updated bundle sources, and a set of deleted bundle names
func (p *Plugin) configDelta(newConfig *Config) (map[string]*Source, map[string]*Source, map[string]struct{}) {
	deletedBundles := map[string]struct{}{}
	for name := range p.config.Bundles {
		deletedBundles[name] = struct{}{}
	}
	newBundles := map[string]*Source{}
	updatedBundles := map[string]*Source{}
	for name, source := range newConfig.Bundles {
		oldSource, found := p.config.Bundles[name]
		if !found {
			newBundles[name] = source
		} else {
			delete(deletedBundles, name)
			if !reflect.DeepEqual(oldSource, source) {
				updatedBundles[name] = source
			}
		}
	}

	return newBundles, updatedBundles, deletedBundles
}

func (p *Plugin) saveBundleToDisk(name string, raw io.Reader) error {

	bundleDir := filepath.Join(p.bundlePersistPath, name)
	tmpFile := filepath.Join(bundleDir, ".bundle.tar.gz.tmp")
	bundleFile := filepath.Join(bundleDir, "bundle.tar.gz")

	saveErr := saveCurrentBundleToDisk(bundleDir, ".bundle.tar.gz.tmp", raw)
	if saveErr != nil {
		p.log(name).Error("Failed to save new bundle to disk: %v", saveErr)

		if err := os.Remove(tmpFile); err != nil {
			p.log(name).Warn("Failed to remove temp file ('%s'): %v", tmpFile, err)
		}

		if _, err := os.Stat(bundleFile); err == nil {
			p.log(name).Warn("Older version of activated bundle persisted, ignoring error")
			return nil
		}
		return saveErr
	}

	return os.Rename(tmpFile, bundleFile)
}

func saveCurrentBundleToDisk(path, filename string, raw io.Reader) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	if raw == nil {
		return fmt.Errorf("no raw bundle bytes to persist to disk")
	}

	dest, err := os.OpenFile(filepath.Join(path, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, raw)
	return err
}

func loadBundleFromDisk(path, name string, src *Source) (*bundle.Bundle, error) {
	bundlePath := filepath.Join(path, name, "bundle.tar.gz")

	if _, err := os.Stat(bundlePath); err == nil {
		f, err := os.Open(filepath.Join(bundlePath))
		if err != nil {
			return nil, err
		}
		defer f.Close()

		r := bundle.NewReader(f)

		if src != nil {
			r = r.WithBundleVerificationConfig(src.Signing)
		}

		b, err := r.Read()
		if err != nil {
			return nil, err
		}
		return &b, nil
	} else if os.IsNotExist(err) {
		return nil, nil
	} else {
		return nil, err
	}
}

func (p *Plugin) log(name string) logging.Logger {
	if p.logger == nil {
		p.logger = logging.Get()
	}
	return p.logger.WithFields(map[string]interface{}{"name": name, "plugin": Name})
}

func (p *Plugin) getBundlePersistPath() (string, error) {
	persistDir, err := p.manager.Config.GetPersistenceDirectory()
	if err != nil {
		return "", err
	}

	return filepath.Join(persistDir, "bundles"), nil
}

type fileLoader struct {
	name           string
	path           string
	bvc            *bundle.VerificationConfig
	sizeLimitBytes int64
	f              func(context.Context, string, download.Update)
}

func (fl *fileLoader) Start(ctx context.Context) {
	go func() {
		fl.oneShot(ctx)
	}()
}

func (*fileLoader) Stop(context.Context) {

}

func (*fileLoader) ClearCache() {

}

func (*fileLoader) SetCache(string) {

}

func (fl *fileLoader) Trigger(ctx context.Context) error {
	fl.oneShot(ctx)
	return nil
}

func (fl *fileLoader) oneShot(ctx context.Context) {
	var u download.Update
	u.Metrics = metrics.New()
	f, err := os.Open(fl.path)
	u.Error = err
	if err != nil {
		fl.f(ctx, fl.name, u)
		return
	}

	defer f.Close()
	b, err := bundle.NewReader(f).
		WithMetrics(u.Metrics).
		WithBundleVerificationConfig(fl.bvc).
		WithSizeLimitBytes(fl.sizeLimitBytes).Read()
	u.Error = err
	if err == nil {
		u.Bundle = &b
	}
	fl.f(ctx, fl.name, u)
}
