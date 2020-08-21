# cell
--
    import "."

Package cell provides a Cell data types to safely read and write, and wait for
concurrent updates from multiple go-routines.

## Usage

#### type Cell

```go
type Cell struct {
}
```

Cell stores some state of type interface{}. A cell must have only a single owner
that reads the state, or waits for state updates, but is allowed to have
multiple setters. In this sense Cell is a multi producer, single consumer
channel. Intermittent updates are lost, in case the cell is updated faster than
the consumer tries to read for state updates. Updates are immediate, there will
be no backpressure applied to producers.

A typical use-case for cell is to generate asynchronous configuration updates
(no deltas).

#### func  NewCell

```go
func NewCell(st interface{}) *Cell
```
NewCell creates a new call instance with its initial state. Subsequent reads
will return this state, if there have been no updates.

#### func (*Cell) Get

```go
func (c *Cell) Get() interface{}
```
Get returns the current state.

#### func (*Cell) Set

```go
func (c *Cell) Set(st interface{})
```
Set updates the state of the Cell and unblocks a waiting consumer. Set does not
block.

#### func (*Cell) Wait

```go
func (c *Cell) Wait(cancel unison.Canceler) (interface{}, error)
```
Wait blocks until it an update since the last call to Get or Wait has been
found. The cancel context can be used to interrupt the call to Wait early. The
error value will be set to the value returned by cancel.Err() in case Wait was
interrupted. Wait does not produce any errors that need to be handled by itself.
