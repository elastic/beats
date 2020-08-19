# management
--
    import "."

Package management provides the integration of the collector with the
elastic-agent via grpc.

## Usage

#### type ConfigManager

```go
type ConfigManager struct {
}
```

ConfigManager establishes a connection with the elastic-agent.

#### func  NewConfigManager

```go
func NewConfigManager(log *logp.Logger, settings Settings) (*ConfigManager, error)
```

#### func (*ConfigManager) Run

```go
func (m *ConfigManager) Run(ctx context.Context, handler EventHandler) error
```

#### func (*ConfigManager) UpdateStatus

```go
func (m *ConfigManager) UpdateStatus(status status.Status, msg string)
```

#### type EventHandler

```go
type EventHandler struct {
	// OnConfig is called when the Elastic Agent is requesting that the configuration
	// be set to the provided new value.
	//
	// XXX: Is this the complete configuration or a delta?
	OnConfig func(*common.Config) error

	OnStop func()
}
```

EventHandler is used to configure callbacks that can be triggered by RPC calls
from the Agent to the Collector.

#### type Settings

```go
type Settings struct {
	Enabled bool   `config:"enabled" yaml:"enabled"`
	Mode    string `config:"mode" yaml:"mode"`
}
```

Settings used to configure the ConfigManager.

#### func (Settings) IsManaged

```go
func (settings Settings) IsManaged() bool
```
