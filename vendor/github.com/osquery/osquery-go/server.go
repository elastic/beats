package osquery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/thrift/lib/go/thrift"

	"github.com/osquery/osquery-go/gen/osquery"
	"github.com/osquery/osquery-go/transport"
	"github.com/pkg/errors"
)

type OsqueryPlugin interface {
	// Name is the name used to refer to the plugin (eg. the name of the
	// table the plugin implements).
	Name() string
	// RegistryName is which "registry" the plugin should be added to.
	// Valid names are ["config", "logger", "table"].
	RegistryName() string
	// Routes returns the detailed information about the interface exposed
	// by the plugin. See the example plugins for samples.
	Routes() osquery.ExtensionPluginResponse
	// Ping implements a health check for the plugin. If the plugin is in a
	// healthy state, StatusOK should be returned.
	Ping() osquery.ExtensionStatus
	// Call requests the plugin to perform its defined behavior, returning
	// a response containing the result.
	Call(context.Context, osquery.ExtensionPluginRequest) osquery.ExtensionResponse
	// Shutdown alerts the plugin to stop.
	Shutdown()
}

const defaultTimeout = 1 * time.Second
const defaultPingInterval = 5 * time.Second

// ExtensionManagerServer is an implementation of the full ExtensionManager
// API. Plugins can register with an extension manager, which handles the
// communication with the osquery process.
type ExtensionManagerServer struct {
	name         string
	sockPath     string
	serverClient ExtensionManager
	registry     map[string](map[string]OsqueryPlugin)
	server       thrift.TServer
	transport    thrift.TServerTransport
	timeout      time.Duration
	pingInterval time.Duration // How often to ping osquery server
	mutex        sync.Mutex
	uuid         osquery.ExtensionRouteUUID
	started      bool // Used to ensure tests wait until the server is actually started
}

// validRegistryNames contains the allowable RegistryName() values. If a plugin
// attempts to register with another value, the program will panic.
var validRegistryNames = map[string]bool{
	"table":       true,
	"logger":      true,
	"config":      true,
	"distributed": true,
}

type ServerOption func(*ExtensionManagerServer)

func ServerTimeout(timeout time.Duration) ServerOption {
	return func(s *ExtensionManagerServer) {
		s.timeout = timeout
	}
}

func ServerPingInterval(interval time.Duration) ServerOption {
	return func(s *ExtensionManagerServer) {
		s.pingInterval = interval
	}
}

// NewExtensionManagerServer creates a new extension management server
// communicating with osquery over the socket at the provided path. If
// resolving the address or connecting to the socket fails, this function will
// error.
func NewExtensionManagerServer(name string, sockPath string, opts ...ServerOption) (*ExtensionManagerServer, error) {
	// Initialize nested registry maps
	registry := make(map[string](map[string]OsqueryPlugin))
	for reg, _ := range validRegistryNames {
		registry[reg] = make(map[string]OsqueryPlugin)
	}

	manager := &ExtensionManagerServer{
		name:         name,
		sockPath:     sockPath,
		registry:     registry,
		timeout:      defaultTimeout,
		pingInterval: defaultPingInterval,
	}

	for _, opt := range opts {
		opt(manager)
	}

	serverClient, err := NewClient(sockPath, manager.timeout)
	if err != nil {
		return nil, err
	}
	manager.serverClient = serverClient

	return manager, nil
}

// RegisterPlugin adds one or more OsqueryPlugins to this extension manager.
func (s *ExtensionManagerServer) RegisterPlugin(plugins ...OsqueryPlugin) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, plugin := range plugins {
		if !validRegistryNames[plugin.RegistryName()] {
			panic("invalid registry name: " + plugin.RegistryName())
		}
		s.registry[plugin.RegistryName()][plugin.Name()] = plugin
	}
}

func (s *ExtensionManagerServer) genRegistry() osquery.ExtensionRegistry {
	registry := osquery.ExtensionRegistry{}
	for regName, _ := range s.registry {
		registry[regName] = osquery.ExtensionRouteTable{}
		for _, plugin := range s.registry[regName] {
			registry[regName][plugin.Name()] = plugin.Routes()
		}
	}
	return registry
}

// Start registers the extension plugins and begins listening on a unix socket
// for requests from the osquery process. All plugins should be registered with
// RegisterPlugin() before calling Start().
func (s *ExtensionManagerServer) Start() error {
	var server thrift.TServer
	err := func() error {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		registry := s.genRegistry()

		stat, err := s.serverClient.RegisterExtension(
			&osquery.InternalExtensionInfo{
				Name: s.name,
			},
			registry,
		)

		if err != nil {
			return errors.Wrap(err, "registering extension")
		}
		if stat.Code != 0 {
			return errors.Errorf("status %d registering extension: %s", stat.Code, stat.Message)
		}
		s.uuid = stat.UUID

		listenPath := fmt.Sprintf("%s.%d", s.sockPath, stat.UUID)

		processor := osquery.NewExtensionProcessor(s)

		s.transport, err = transport.OpenServer(listenPath, s.timeout)
		if err != nil {
			openError := errors.Wrapf(err, "opening server socket (%s)", listenPath)
			_, err = s.serverClient.DeregisterExtension(stat.UUID)
			if err != nil {
				return errors.Wrapf(err, "deregistering extension - follows %s", openError.Error())
			}
			return openError
		}

		s.server = thrift.NewTSimpleServer2(processor, s.transport)
		server = s.server

		s.started = true

		return nil
	}()

	if err != nil {
		return err
	}

	return server.Serve()
}

// Run starts the extension manager and runs until osquery calls for a shutdown
// or the osquery instance goes away.
func (s *ExtensionManagerServer) Run() error {
	errc := make(chan error)
	go func() {
		errc <- s.Start()
	}()

	// Watch for the osquery process going away. If so, initiate shutdown.
	go func() {
		for {
			time.Sleep(s.pingInterval)

			status, err := s.serverClient.Ping()
			if err != nil {
				errc <- errors.Wrap(err, "extension ping failed")
				break
			}
			if status.Code != 0 {
				errc <- errors.Errorf("ping returned status %d", status.Code)
				break
			}
		}
	}()

	err := <-errc
	if err := s.Shutdown(context.Background()); err != nil {
		return err
	}
	return err
}

// Ping implements the basic health check.
func (s *ExtensionManagerServer) Ping(ctx context.Context) (*osquery.ExtensionStatus, error) {
	return &osquery.ExtensionStatus{Code: 0, Message: "OK"}, nil
}

// Call routes a call from the osquery process to the appropriate registered
// plugin.
func (s *ExtensionManagerServer) Call(ctx context.Context, registry string, item string, request osquery.ExtensionPluginRequest) (*osquery.ExtensionResponse, error) {
	subreg, ok := s.registry[registry]
	if !ok {
		return &osquery.ExtensionResponse{
			Status: &osquery.ExtensionStatus{
				Code:    1,
				Message: "Unknown registry: " + registry,
			},
		}, nil
	}

	plugin, ok := subreg[item]
	if !ok {
		return &osquery.ExtensionResponse{
			Status: &osquery.ExtensionStatus{
				Code:    1,
				Message: "Unknown registry item: " + item,
			},
		}, nil
	}

	response := plugin.Call(context.Background(), request)
	return &response, nil
}

// Shutdown deregisters the extension, stops the server and closes all sockets.
func (s *ExtensionManagerServer) Shutdown(ctx context.Context) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stat, err := s.serverClient.DeregisterExtension(s.uuid)
	err = errors.Wrap(err, "deregistering extension")
	if err == nil && stat.Code != 0 {
		err = errors.Errorf("status %d deregistering extension: %s", stat.Code, stat.Message)
	}
	s.serverClient.Close()
	if s.server != nil {
		server := s.server
		s.server = nil
		// Stop the server asynchronously so that the current request
		// can complete. Otherwise, this is vulnerable to deadlock if a
		// shutdown request is being processed when shutdown is
		// explicitly called.
		go func() {
			server.Stop()
		}()
	}

	return
}

// Useful for testing
func (s *ExtensionManagerServer) waitStarted() {
	for {
		s.mutex.Lock()
		started := s.started
		s.mutex.Unlock()
		if started {
			time.Sleep(10 * time.Millisecond)
			break
		}
	}
}
