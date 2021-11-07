// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package plugins implements plugin management for the policy engine.
package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
	"github.com/open-policy-agent/opa/config"
	bundleUtils "github.com/open-policy-agent/opa/internal/bundle"
	cfg "github.com/open-policy-agent/opa/internal/config"
	initload "github.com/open-policy-agent/opa/internal/runtime/init"
	"github.com/open-policy-agent/opa/keys"
	"github.com/open-policy-agent/opa/loader"
	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/plugins/rest"
	"github.com/open-policy-agent/opa/resolver/wasm"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/topdown/cache"
	"github.com/open-policy-agent/opa/topdown/print"
)

// Factory defines the interface OPA uses to instantiate your plugin.
//
// When OPA processes it's configuration it looks for factories that
// have been registered by calling runtime.RegisterPlugin. Factories
// are registered to a name which is used to key into the
// configuration blob. If your plugin has not been configured, your
// factory will not be invoked.
//
//   plugins:
//     my_plugin1:
//       some_key: foo
//     # my_plugin2:
//     #   some_key2: bar
//
// If OPA was started with the configuration above and received two
// calls to runtime.RegisterPlugins (one with NAME "my_plugin1" and
// one with NAME "my_plugin2"), it would only invoke the factory for
// for my_plugin1.
//
// OPA instantiates and reconfigures plugins in two steps. First, OPA
// will call Validate to check the configuration. Assuming the
// configuration is valid, your factory should return a configuration
// value that can be used to construct your plugin. Second, OPA will
// call New to instantiate your plugin providing the configuration
// value returned from the Validate call.
//
// Validate receives a slice of bytes representing plugin
// configuration and returns a configuration value that can be used to
// instantiate your plugin. The manager is provided to give access to
// the OPA's compiler, storage layer, and global configuration. Your
// Validate function will typically:
//
//  1. Deserialize the raw config bytes
//  2. Validate the deserialized config for semantic errors
//  3. Inject default values
//  4. Return a deserialized/parsed config
//
// New receives a valid configuration for your plugin and returns a
// plugin object. Your New function will typically:
//
//  1. Cast the config value to it's own type
//  2. Instantiate a plugin object
//  3. Return the plugin object
//  4. Update status via `plugins.Manager#UpdatePluginStatus`
//
// After a plugin has been created subsequent status updates can be
// send anytime the plugin enters a ready or error state.
type Factory interface {
	Validate(manager *Manager, config []byte) (interface{}, error)
	New(manager *Manager, config interface{}) Plugin
}

// Plugin defines the interface OPA uses to manage your plugin.
//
// When OPA starts it will start all of the plugins it was configured
// to instantiate. Each time a new plugin is configured (via
// discovery), OPA will start it. You can use the Start call to spawn
// additional goroutines or perform initialization tasks.
//
// Currently OPA will not call Stop on plugins.
//
// When OPA receives new configuration for your plugin via discovery
// it will first Validate the configuration using your factory and
// then call Reconfigure.
type Plugin interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context)
	Reconfigure(ctx context.Context, config interface{})
}

// Triggerable defines the interface plugins use for manual plugin triggers.
type Triggerable interface {
	Trigger(context.Context) error
}

// State defines the state that a Plugin instance is currently
// in with pre-defined states.
type State string

const (
	// StateNotReady indicates that the Plugin is not in an error state, but isn't
	// ready for normal operation yet. This should only happen at
	// initialization time.
	StateNotReady State = "NOT_READY"

	// StateOK signifies that the Plugin is operating normally.
	StateOK State = "OK"

	// StateErr indicates that the Plugin is in an error state and should not
	// be considered as functional.
	StateErr State = "ERROR"

	// StateWarn indicates the Plugin is operating, but in a potentially dangerous or
	// degraded state. It may be used to indicate manual remediation is needed, or to
	// alert admins of some other noteworthy state.
	StateWarn State = "WARN"
)

// TriggerMode defines the trigger mode utilized by a Plugin for bundle download,
// log upload etc.
type TriggerMode string

const (
	// TriggerPeriodic represents periodic polling mechanism
	TriggerPeriodic TriggerMode = "periodic"

	// TriggerManual represents manual triggering mechanism
	TriggerManual TriggerMode = "manual"

	// DefaultTriggerMode represents default trigger mechanism
	DefaultTriggerMode TriggerMode = "periodic"
)

// Status has a Plugin's current status plus an optional Message.
type Status struct {
	State   State  `json:"state"`
	Message string `json:"message,omitempty"`
}

func (s *Status) String() string {
	return fmt.Sprintf("{%v %q}", s.State, s.Message)
}

// StatusListener defines a handler to register for status updates.
type StatusListener func(status map[string]*Status)

// Manager implements lifecycle management of plugins and gives plugins access
// to engine-wide components like storage.
type Manager struct {
	Store  storage.Store
	Config *config.Config
	Info   *ast.Term
	ID     string

	compiler                     *ast.Compiler
	compilerMux                  sync.RWMutex
	wasmResolvers                []*wasm.Resolver
	wasmResolversMtx             sync.RWMutex
	services                     map[string]rest.Client
	keys                         map[string]*keys.Config
	plugins                      []namedplugin
	registeredTriggers           []func(txn storage.Transaction)
	mtx                          sync.Mutex
	pluginStatus                 map[string]*Status
	pluginStatusListeners        map[string]StatusListener
	initBundles                  map[string]*bundle.Bundle
	initFiles                    loader.Result
	maxErrors                    int
	initialized                  bool
	interQueryBuiltinCacheConfig *cache.Config
	gracefulShutdownPeriod       int
	registeredCacheTriggers      []func(*cache.Config)
	logger                       logging.Logger
	consoleLogger                logging.Logger
	serverInitialized            chan struct{}
	serverInitializedOnce        sync.Once
	printHook                    print.Hook
	enablePrintStatements        bool
}

type managerContextKey string
type managerWasmResolverKey string

const managerCompilerContextKey = managerContextKey("compiler")
const managerWasmResolverContextKey = managerWasmResolverKey("wasmResolvers")

// SetCompilerOnContext puts the compiler into the storage context. Calling this
// function before committing updated policies to storage allows the manager to
// skip parsing and compiling of modules. Instead, the manager will use the
// compiler that was stored on the context.
func SetCompilerOnContext(context *storage.Context, compiler *ast.Compiler) {
	context.Put(managerCompilerContextKey, compiler)
}

// GetCompilerOnContext gets the compiler cached on the storage context.
func GetCompilerOnContext(context *storage.Context) *ast.Compiler {
	compiler, ok := context.Get(managerCompilerContextKey).(*ast.Compiler)
	if !ok {
		return nil
	}
	return compiler
}

// SetWasmResolversOnContext puts a set of Wasm Resolvers into the storage
// context. Calling this function before committing updated wasm modules to
// storage allows the manager to skip initializing modules before using them.
// Instead, the manager will use the compiler that was stored on the context.
func SetWasmResolversOnContext(context *storage.Context, rs []*wasm.Resolver) {
	context.Put(managerWasmResolverContextKey, rs)
}

// getWasmResolversOnContext gets the resolvers cached on the storage context.
func getWasmResolversOnContext(context *storage.Context) []*wasm.Resolver {
	resolvers, ok := context.Get(managerWasmResolverContextKey).([]*wasm.Resolver)
	if !ok {
		return nil
	}
	return resolvers
}

func validateTriggerMode(mode TriggerMode) error {
	switch mode {
	case TriggerPeriodic, TriggerManual:
		return nil
	default:
		return fmt.Errorf("invalid trigger mode %q (want %q or %q)", mode, TriggerPeriodic, TriggerManual)
	}
}

// ValidateAndInjectDefaultsForTriggerMode validates the trigger mode and injects default values
func ValidateAndInjectDefaultsForTriggerMode(a, b *TriggerMode) (*TriggerMode, error) {

	if a == nil && b != nil {
		err := validateTriggerMode(*b)
		if err != nil {
			return nil, err
		}
		return b, nil
	} else if a != nil && b == nil {
		err := validateTriggerMode(*a)
		if err != nil {
			return nil, err
		}
		return a, nil
	} else if a != nil && b != nil {
		if *a != *b {
			return nil, fmt.Errorf("trigger mode mismatch: %s and %s (hint: check discovery configuration)", *a, *b)
		}
		err := validateTriggerMode(*a)
		if err != nil {
			return nil, err
		}
		return a, nil

	} else {
		t := DefaultTriggerMode
		return &t, nil
	}
}

type namedplugin struct {
	name   string
	plugin Plugin
}

// Info sets the runtime information on the manager. The runtime information is
// propagated to opa.runtime() built-in function calls.
func Info(term *ast.Term) func(*Manager) {
	return func(m *Manager) {
		m.Info = term
	}
}

// InitBundles provides the initial set of bundles to load.
func InitBundles(b map[string]*bundle.Bundle) func(*Manager) {
	return func(m *Manager) {
		m.initBundles = b
	}
}

// InitFiles provides the initial set of other data/policy files to load.
func InitFiles(f loader.Result) func(*Manager) {
	return func(m *Manager) {
		m.initFiles = f
	}
}

// MaxErrors sets the error limit for the manager's shared compiler.
func MaxErrors(n int) func(*Manager) {
	return func(m *Manager) {
		m.maxErrors = n
	}
}

// GracefulShutdownPeriod passes the configured graceful shutdown period to plugins
func GracefulShutdownPeriod(gracefulShutdownPeriod int) func(*Manager) {
	return func(m *Manager) {
		m.gracefulShutdownPeriod = gracefulShutdownPeriod
	}
}

// Logger configures the passed logger on the plugin manager (useful to
// configure default fields)
func Logger(logger logging.Logger) func(*Manager) {
	return func(m *Manager) {
		m.logger = logger
	}
}

// ConsoleLogger sets the passed logger to be used by plugins that are
// configured with console logging enabled.
func ConsoleLogger(logger logging.Logger) func(*Manager) {
	return func(m *Manager) {
		m.consoleLogger = logger
	}
}

func EnablePrintStatements(yes bool) func(*Manager) {
	return func(m *Manager) {
		m.enablePrintStatements = yes
	}
}

func PrintHook(h print.Hook) func(*Manager) {
	return func(m *Manager) {
		m.printHook = h
	}
}

// New creates a new Manager using config.
func New(raw []byte, id string, store storage.Store, opts ...func(*Manager)) (*Manager, error) {

	parsedConfig, err := config.ParseConfig(raw, id)
	if err != nil {
		return nil, err
	}

	keys, err := keys.ParseKeysConfig(parsedConfig.Keys)
	if err != nil {
		return nil, err
	}

	interQueryBuiltinCacheConfig, err := cache.ParseCachingConfig(parsedConfig.Caching)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		Store:                        store,
		Config:                       parsedConfig,
		ID:                           id,
		keys:                         keys,
		pluginStatus:                 map[string]*Status{},
		pluginStatusListeners:        map[string]StatusListener{},
		maxErrors:                    -1,
		interQueryBuiltinCacheConfig: interQueryBuiltinCacheConfig,
		serverInitialized:            make(chan struct{}),
	}

	if m.logger == nil {
		m.logger = logging.Get()
	}

	if m.consoleLogger == nil {
		m.consoleLogger = logging.New()
	}

	serviceOpts := cfg.ServiceOptions{
		Raw:        parsedConfig.Services,
		AuthPlugin: m.AuthPlugin,
		Keys:       keys,
		Logger:     m.logger,
	}
	services, err := cfg.ParseServicesConfig(serviceOpts)
	if err != nil {
		return nil, err
	}

	m.services = services

	for _, f := range opts {
		f(m)
	}

	return m, nil
}

// Init returns an error if the manager could not initialize itself. Init() should
// be called before Start(). Init() is idempotent.
func (m *Manager) Init(ctx context.Context) error {

	if m.initialized {
		return nil
	}

	params := storage.TransactionParams{
		Write:   true,
		Context: storage.NewContext(),
	}

	err := storage.Txn(ctx, m.Store, params, func(txn storage.Transaction) error {

		result, err := initload.InsertAndCompile(ctx, initload.InsertAndCompileOptions{
			Store:                 m.Store,
			Txn:                   txn,
			Files:                 m.initFiles,
			Bundles:               m.initBundles,
			MaxErrors:             m.maxErrors,
			EnablePrintStatements: m.enablePrintStatements,
		})

		if err != nil {
			return err
		}

		SetCompilerOnContext(params.Context, result.Compiler)

		resolvers, err := bundleUtils.LoadWasmResolversFromStore(ctx, m.Store, txn, nil)
		if err != nil {
			return err
		}
		SetWasmResolversOnContext(params.Context, resolvers)

		_, err = m.Store.Register(ctx, txn, storage.TriggerConfig{OnCommit: m.onCommit})
		return err
	})

	if err != nil {
		return err
	}

	m.initialized = true
	return nil
}

// Labels returns the set of labels from the configuration.
func (m *Manager) Labels() map[string]string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.Config.Labels
}

// InterQueryBuiltinCacheConfig returns the configuration for the inter-query cache.
func (m *Manager) InterQueryBuiltinCacheConfig() *cache.Config {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.interQueryBuiltinCacheConfig
}

// Register adds a plugin to the manager. When the manager is started, all of
// the plugins will be started.
func (m *Manager) Register(name string, plugin Plugin) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.plugins = append(m.plugins, namedplugin{
		name:   name,
		plugin: plugin,
	})
	if _, ok := m.pluginStatus[name]; !ok {
		m.pluginStatus[name] = &Status{State: StateNotReady}
	}
}

// Plugins returns the list of plugins registered with the manager.
func (m *Manager) Plugins() []string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	result := make([]string, len(m.plugins))
	for i := range m.plugins {
		result[i] = m.plugins[i].name
	}
	return result
}

// Plugin returns the plugin registered with name or nil if name is not found.
func (m *Manager) Plugin(name string) Plugin {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for i := range m.plugins {
		if m.plugins[i].name == name {
			return m.plugins[i].plugin
		}
	}
	return nil
}

// AuthPlugin returns the HTTPAuthPlugin registered with name or nil if name is not found.
func (m *Manager) AuthPlugin(name string) rest.HTTPAuthPlugin {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for i := range m.plugins {
		if m.plugins[i].name == name {
			return m.plugins[i].plugin.(rest.HTTPAuthPlugin)
		}
	}
	return nil
}

// GetCompiler returns the manager's compiler.
func (m *Manager) GetCompiler() *ast.Compiler {
	m.compilerMux.RLock()
	defer m.compilerMux.RUnlock()
	return m.compiler
}

func (m *Manager) setCompiler(compiler *ast.Compiler) {
	m.compilerMux.Lock()
	defer m.compilerMux.Unlock()
	m.compiler = compiler
}

// RegisterCompilerTrigger registers for change notifications when the compiler
// is changed.
func (m *Manager) RegisterCompilerTrigger(f func(txn storage.Transaction)) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.registeredTriggers = append(m.registeredTriggers, f)
}

// GetWasmResolvers returns the manager's set of Wasm Resolvers.
func (m *Manager) GetWasmResolvers() []*wasm.Resolver {
	m.wasmResolversMtx.RLock()
	defer m.wasmResolversMtx.RUnlock()
	return m.wasmResolvers
}

func (m *Manager) setWasmResolvers(rs []*wasm.Resolver) {
	m.wasmResolversMtx.Lock()
	defer m.wasmResolversMtx.Unlock()
	m.wasmResolvers = rs
}

// Start starts the manager. Init() should be called once before Start().
func (m *Manager) Start(ctx context.Context) error {

	if m == nil {
		return nil
	}

	if !m.initialized {
		if err := m.Init(ctx); err != nil {
			return err
		}
	}

	var toStart []Plugin

	func() {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		toStart = make([]Plugin, len(m.plugins))
		for i := range m.plugins {
			toStart[i] = m.plugins[i].plugin
		}
	}()

	for i := range toStart {
		if err := toStart[i].Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop stops the manager, stopping all the plugins registered with it. Any plugin that needs to perform cleanup should
// do so within the duration of the graceful shutdown period passed with the context as a timeout.
func (m *Manager) Stop(ctx context.Context) {
	var toStop []Plugin

	func() {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		toStop = make([]Plugin, len(m.plugins))
		for i := range m.plugins {
			toStop[i] = m.plugins[i].plugin
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.gracefulShutdownPeriod)*time.Second)
	defer cancel()
	for i := range toStop {
		toStop[i].Stop(ctx)
	}
}

// Reconfigure updates the configuration on the manager.
func (m *Manager) Reconfigure(config *config.Config) error {
	opts := cfg.ServiceOptions{
		Raw:        config.Services,
		AuthPlugin: m.AuthPlugin,
		Logger:     m.logger,
	}

	keys, err := keys.ParseKeysConfig(config.Keys)
	if err != nil {
		return err
	}
	opts.Keys = keys

	services, err := cfg.ParseServicesConfig(opts)
	if err != nil {
		return err
	}

	interQueryBuiltinCacheConfig, err := cache.ParseCachingConfig(config.Caching)
	if err != nil {
		return err
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	config.Labels = m.Config.Labels // don't overwrite labels
	m.Config = config
	m.interQueryBuiltinCacheConfig = interQueryBuiltinCacheConfig
	for name, client := range services {
		m.services[name] = client
	}

	for name, key := range keys {
		m.keys[name] = key
	}

	for _, trigger := range m.registeredCacheTriggers {
		trigger(interQueryBuiltinCacheConfig)
	}

	return nil
}

// PluginStatus returns the current statuses of any plugins registered.
func (m *Manager) PluginStatus() map[string]*Status {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return m.copyPluginStatus()
}

// RegisterPluginStatusListener registers a StatusListener to be
// called when plugin status updates occur.
func (m *Manager) RegisterPluginStatusListener(name string, listener StatusListener) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.pluginStatusListeners[name] = listener
}

// UnregisterPluginStatusListener removes a StatusListener registered with the
// same name.
func (m *Manager) UnregisterPluginStatusListener(name string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	delete(m.pluginStatusListeners, name)
}

// UpdatePluginStatus updates a named plugins status. Any registered
// listeners will be called with a copy of the new state of all
// plugins.
func (m *Manager) UpdatePluginStatus(pluginName string, status *Status) {

	var toNotify map[string]StatusListener
	var statuses map[string]*Status

	func() {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		m.pluginStatus[pluginName] = status
		toNotify = make(map[string]StatusListener, len(m.pluginStatusListeners))
		for k, v := range m.pluginStatusListeners {
			toNotify[k] = v
		}
		statuses = m.copyPluginStatus()
	}()

	for _, l := range toNotify {
		l(statuses)
	}
}

func (m *Manager) copyPluginStatus() map[string]*Status {
	statusCpy := map[string]*Status{}
	for k, v := range m.pluginStatus {
		var cpy *Status
		if v != nil {
			cpy = &Status{
				State:   v.State,
				Message: v.Message,
			}
		}
		statusCpy[k] = cpy
	}
	return statusCpy
}

func (m *Manager) onCommit(ctx context.Context, txn storage.Transaction, event storage.TriggerEvent) {

	compiler := GetCompilerOnContext(event.Context)

	// If the context does not contain the compiler fallback to loading the
	// compiler from the store. Currently the bundle plugin sets the
	// compiler on the context but the server does not (nor would users
	// implementing their own policy loading.)
	if compiler == nil && event.PolicyChanged() {
		compiler, _ = loadCompilerFromStore(ctx, m.Store, txn, m.enablePrintStatements)
	}

	if compiler != nil {
		m.setCompiler(compiler)
		for _, f := range m.registeredTriggers {
			f(txn)
		}
	}

	// Similar to the compiler, look for a set of resolvers on the transaction
	// context. If they are not set we may need to reload from the store.
	resolvers := getWasmResolversOnContext(event.Context)
	if resolvers != nil {
		m.setWasmResolvers(resolvers)

	} else if event.DataChanged() {
		if requiresWasmResolverReload(event) {
			resolvers, err := bundleUtils.LoadWasmResolversFromStore(ctx, m.Store, txn, nil)
			if err != nil {
				panic(err)
			}
			m.setWasmResolvers(resolvers)
		} else {
			err := m.updateWasmResolversData(ctx, event)
			if err != nil {
				panic(err)
			}
		}
	}
}

func loadCompilerFromStore(ctx context.Context, store storage.Store, txn storage.Transaction, enablePrintStatements bool) (*ast.Compiler, error) {
	policies, err := store.ListPolicies(ctx, txn)
	if err != nil {
		return nil, err
	}
	modules := map[string]*ast.Module{}

	for _, policy := range policies {
		bs, err := store.GetPolicy(ctx, txn, policy)
		if err != nil {
			return nil, err
		}
		module, err := ast.ParseModule(policy, string(bs))
		if err != nil {
			return nil, err
		}
		modules[policy] = module
	}

	compiler := ast.NewCompiler().WithEnablePrintStatements(enablePrintStatements)
	compiler.Compile(modules)
	return compiler, nil
}

func requiresWasmResolverReload(event storage.TriggerEvent) bool {
	// If the data changes touched the bundle path (which includes
	// the wasm modules) we will reload them. Otherwise update
	// data for each module already on the manager.
	for _, dataEvent := range event.Data {
		if dataEvent.Path.HasPrefix(bundle.BundlesBasePath) {
			return true
		}
	}
	return false
}

func (m *Manager) updateWasmResolversData(ctx context.Context, event storage.TriggerEvent) error {
	m.wasmResolversMtx.Lock()
	defer m.wasmResolversMtx.Unlock()

	if len(m.wasmResolvers) == 0 {
		return nil
	}

	for _, resolver := range m.wasmResolvers {
		for _, dataEvent := range event.Data {
			var err error
			if dataEvent.Removed {
				err = resolver.RemoveDataPath(ctx, dataEvent.Path)
			} else {
				err = resolver.SetDataPath(ctx, dataEvent.Path, dataEvent.Data)
			}
			if err != nil {
				return fmt.Errorf("failed to update wasm runtime data: %s", err)
			}
		}
	}
	return nil
}

// PublicKeys returns a public keys that can be used for verifying signed bundles.
func (m *Manager) PublicKeys() map[string]*keys.Config {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.keys
}

// Client returns a client for communicating with a remote service.
func (m *Manager) Client(name string) rest.Client {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.services[name]
}

// Services returns a list of services that m can provide clients for.
func (m *Manager) Services() []string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	s := make([]string, 0, len(m.services))
	for name := range m.services {
		s = append(s, name)
	}
	return s
}

// Logger gets the standard logger for this plugin manager.
func (m *Manager) Logger() logging.Logger {
	return m.logger
}

// ConsoleLogger gets the console logger for this plugin manager.
func (m *Manager) ConsoleLogger() logging.Logger {
	return m.consoleLogger
}

func (m *Manager) PrintHook() print.Hook {
	return m.printHook
}

func (m *Manager) EnablePrintStatements() bool {
	return m.enablePrintStatements
}

// ServerInitialized signals a channel indicating that the OPA
// server has finished initialization.
func (m *Manager) ServerInitialized() {
	m.serverInitializedOnce.Do(func() { close(m.serverInitialized) })
}

// ServerInitializedChannel returns a receive-only channel that
// is closed when the OPA server has finished initialization.
// Be aware that the socket of the server listener may not be
// open by the time this channel is closed. There is a very
// small window where the socket may still be closed, due to
// a race condition.
func (m *Manager) ServerInitializedChannel() <-chan struct{} {
	return m.serverInitialized
}

// RegisterCacheTrigger accepts a func that receives new inter-query cache config generated by
// a reconfigure of the plugin manager, so that it can be propagated to existing inter-query caches.
func (m *Manager) RegisterCacheTrigger(trigger func(*cache.Config)) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.registeredCacheTriggers = append(m.registeredCacheTriggers, trigger)
}
