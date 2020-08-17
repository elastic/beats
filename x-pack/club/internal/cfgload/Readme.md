# cfgload
--
    import "."

Package cfgload provides support for reading configuration files from disk.

## Usage

#### type Loader

```go
type Loader struct {
	Home              string
	StrictPermissions bool
}
```

Loader is used to configuration files.

#### func (*Loader) ReadFiles

```go
func (r *Loader) ReadFiles(files []string) (*common.Config, error)
```
ReadFiles reads and merges the configurations provided by the files slice. Load
order depends on the the files are passed in. Settings in later files overwrite
already existing settings.

#### type Reader

```go
type Reader interface {
	ReadFiles(files []string) (*common.Config, error)
}
```


#### type Watcher

```go
type Watcher struct {
	Log    *logp.Logger
	Files  []string
	Reader Reader
}
```

Watcher monitors the paths given in Files for changes.

#### func (*Watcher) Run

```go
func (w *Watcher) Run(cancel unison.Canceler, handler func(*common.Config) error) error
```
Run executes the watchers main loop. It blocks until the watcher is shut down.
The handler function is called with the merged configuration if the watcher
detects any file changes.
