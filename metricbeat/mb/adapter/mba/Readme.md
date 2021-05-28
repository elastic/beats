# mba
--
    import "."

Package mba provides adapters for using metricbeat modules and metricsets as v2
inputs.

The metricsets provided by these wrappers are mostly independent of the
Metricbeat core framework. No globals or shared state in the metricbeat core
will be read or written. Each MeticsetManager will be fully independent.

The MetricsetManager is used to wrap a Metricbeat module and its Metricsets.

In Metricbeat a module implementation is optional. The module implementations
purpose is to provide additional functionality (parsing, query, cache data,
coordination/sharing) for its metricsets. A module instance will be shared
between all metricsets. Independent if a Module is implemnted or not, an
mb.Module instance will be passed to the metricset.

The adapters provided in this package provide similar functionality to
metricbeat. The sharing of Modules with metricsets requires authors to create a
common ModuleAdapter, that will be shared between add metricset input adapters.

Note: The ModuleAdapter should also be used for metricsets that do not require a
shared module.

Example system module:

```

func systemMetricsPlugins() []v2.Plugin {

    // Create shared module adapter. The Factory is optional. The adapter
    // provides helpers to create inputs from metricsets.
    systemModule := &mba.ModuleAdapter{Name: "system", Factory: system.NewModule}

    // Create list of inputs from metricset implementations:
    return []v2.Plugin{
    		systemModule.MetricsetInput("system.cpu", "cpu", cpu.New),
    		...
    }

}

```

Metricbeat allows developers to pass additional options when registering with
the mb.Registry. The optionas can provide simple meta-data, some form of config
validation, or additional hooks to modify some of the default behavior (e.g.
HostParser). The MetricsetManager returned by (*ModuleAdapter).MetricsetInput
can be modified directly, or using one of its WithX methods.

## Usage

#### func  Plugin

```go
func Plugin(stability feature.Stability, deprecated bool, mm MetricsetManager) v2.Plugin
```
Plugin create a v2.Plugin for a MetricsetManager.

#### type MetricsetManager

```go
type MetricsetManager struct {
	MetricsetName string
	InputName     string
	ModuleManager ModuleManager

	Factory mb.MetricSetFactory

	HostParser mb.HostParser
	Namespace  string
}
```

MetricsetManager provides the v2.InputManager that will provide a Metricset as
an v2.Input.

#### func (*MetricsetManager) Create

```go
func (m *MetricsetManager) Create(cfg *common.Config) (v2.Input, error)
```
Create builds a new Input instance from the given configuation, or returns an
error if the configuation is invalid. The input must establish any connection
for data collection yet. The Beat will use the Test/Run methods of the input.

#### func (*MetricsetManager) Init

```go
func (m *MetricsetManager) Init(grp unison.Group, mode v2.Mode) error
```

#### func (MetricsetManager) WithHostParser

```go
func (m MetricsetManager) WithHostParser(p mb.HostParser) MetricsetManager
```
WithHostParser creates a new MetricsetManager using the new host parser.

#### func (MetricsetManager) WithNamespace

```go
func (m MetricsetManager) WithNamespace(n string) MetricsetManager
```
WithNamespace creates a new MetricsetManager using the new namespace.

#### type ModuleAdapter

```go
type ModuleAdapter struct {
	Name      string
	Factory   mb.ModuleFactory
	Modifiers []mb.EventModifier
}
```


#### func (*ModuleAdapter) Create

```go
func (ma *ModuleAdapter) Create(base mb.BaseModule) (mb.Module, error)
```

#### func (*ModuleAdapter) EventModifiers

```go
func (ma *ModuleAdapter) EventModifiers() []mb.EventModifier
```

#### func (*ModuleAdapter) MetricsetInput

```go
func (ma *ModuleAdapter) MetricsetInput(inputName string, metricsetName string, factory mb.MetricSetFactory) MetricsetManager
```

#### func (*ModuleAdapter) ModuleName

```go
func (ma *ModuleAdapter) ModuleName() string
```

#### type ModuleManager

```go
type ModuleManager interface {
	ModuleName() string
	Create(base mb.BaseModule) (mb.Module, error)
	EventModifiers() []mb.EventModifier
}
```

ModuleManager interfaces that can provide a metricset with a custom mb.Module at
initialization time. Normally one want to use ModuleAdapter, in order to wrap an
existing Module, or create a ModuleManager with dedicated Module implementation.
