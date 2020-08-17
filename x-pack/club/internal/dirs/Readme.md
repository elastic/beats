# dirs
--
    import "."

Package dirs provides an unified view on disk layout for common directories.

## Usage

#### type Project

```go
type Project struct {
	Home   string
	Config string
	Data   string
	Logs   string
}
```

Project lists common directory types an application would require for its
internal use. Use any of the ProjectFromX functions to initialize project.

#### func  ProjectFrom

```go
func ProjectFrom(home string) (Project, error)
```
ProjectFrom initializes a Project directory listing, using path as the home
directory. The home directory layout is assumed to follow the elastic common
directory layout. The current working directory will be used if home is not
empty.

#### func  ProjectFromCWD

```go
func ProjectFromCWD() (Project, error)
```
ProjectFromCWD initializes a Project, assuming the elastic common directory
layout.

#### func (Project) Update

```go
func (p Project) Update(updates Project) Project
```
Update overwrites paths using the list of updates. Empty entries in updates are
ignored.
