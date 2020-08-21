# registries
--
    import "."

Package registries provides utility functions for wrapping beats functionality
into v2.Registries in order to discovery and create inputs based on legacy code.

## Usage

#### func  Combine

```go
func Combine(registries ...v2.Registry) v2.Registry
```
Combine combines a list of input registries into a single registry. When
configuring an input the each registry is tried. The first registry that returns
an input type wins. registry in the list should have a type prefix to allow some
routing.

The registryList can be used to combine v2 style inputs and old RunnerFactory
into a single namespace. By listing v2 style inputs first we can shadow older
implementations without fully replacing them in the Beats code-base.

#### func  Prefixed

```go
func Prefixed(name string, reg v2.Registry) v2.Registry
```
Prefixed wraps a Registry into a prefixRegistry. All inputs in the input
registry are now addressable via the common prefix only. For example this setup:

    reg = withTypePrefix("logs", filebeatInputs)

requires the input configuration to load the journald input like this:

    - type: logs/journald

#### type RunnerFactoryRegistry

```go
type RunnerFactoryRegistry struct {
	TypeField string
	Factory   cfgfile.RunnerFactory
	Has       func(string) bool
}
```

RunnerFactoryRegistry wraps a runner factory and makes it available with the
filebeat v2 input API. Config validation is best effort and needs to be defered
for until the input is actually run. We can't tell for sure in advance if the
input type exists when the plugin is configured. Some beats allow some
introspection of existing input types, which can be exposed to the
runnerFactoryRegistry by implementing has.

#### func (*RunnerFactoryRegistry) Find

```go
func (r *RunnerFactoryRegistry) Find(name string) (v2.Plugin, bool)
```

#### func (*RunnerFactoryRegistry) Init

```go
func (r *RunnerFactoryRegistry) Init(_ unison.Group, _ v2.Mode) error
```
