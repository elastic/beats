# beatsout
--
    import "."

Package beatsouts allows the reuse of existing libbeat based outputs.

TODO: The packag currently wraps libbeat outputs by accessing the outputs
registry.

    It would be better to allow developers to create wrappers more selectively, such that
    configuration rewrites are possible.

## Usage

#### func  NewOutputFactory

```go
func NewOutputFactory(info beat.Info) publishing.OutputFactory
```
NewOutputFactory creates a new publishing.OutputFactory, that can be used to
create outputs based on existing libbeat outputs.

When creating an output we create a complete libbeat publisher pipeline
including queue, ack handling and actual libbeat outputs for publishing the
events. The pipeline is wrapped, such that is satifies the publishing.Output
interface.
