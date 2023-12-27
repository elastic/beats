# Beats Proxy Queue

The proxy queue is an implementation of the [beats Queue interface](https://github.com/elastic/beats/blob/main/libbeat/publisher/queue/queue.go) meant to work with the Shipper output. The Shipper output is unique because rather than sending events to a remote server it sends them to the Shipper, a local process that has its own queue where it stages events for delivery to their true destination upstream. This means that if the Shipper output is used with a conventional queue, events will remain queued in both Beats _and_ the shipper until they receive upstream acknowledgment, potentially doubling the memory needed for a given set of events.

The solution to this is the proxy queue: from the perspective of the Beats pipeline, it behaves like a normal (albeit small) queue, but its buffer is immediately cleared on being read, and it provides a hook in its event batches for the output to free its contents once sent, while still preserving metadata so that inputs that require end-to-end acknowledgment of their events can receive the acknowledgments later, after the Shipper confirms upstream ingestion.

## Limitations

Some features present in other queues are unimplemented or ignored by the proxy queue since they are unneeded when ingesting via the Shipper output:

- `queue.EntryID`: a `Publish` call to a normal queue returns an `EntryID`, a unique integer that is incremented with each event. This data is only used internally in the Shipper to track acknowledgments, and is unused by Beats.
- Producer cancel: When a `queue.Producer` (the API interface for adding data to a queue) is cancelled, the memory queue attempts to remove from its buffer any events sent by that producer that have not yet been consumed. This feature is only ever used during Beats shutdown, and since the proxy queue in particular never accumulates events itself but instead stores them in the Shipper's queue, it has no mechanism to cancel most outstanding events.
- Requested batch size: The queue interface reads event batches by specifying the desired number of events, which the queue will try to satisfy. Because batches from the proxy queue are being sent to a local process rather than over a network, there is less performance sensitivity to the batch size. Because the proxy queue optimizes its buffer by using it to directly store the batch contents, we can get simpler and more consistent performance by accumulating up to a maximum size and then sending that immediately when a batch is requested. Therefore the proxy queue has its own configurable target batch size, and ignores the parameter given by the consumer.
- Metrics: The proxy queue implements the usual queue metrics for the Beats pipeline, however it doesn't implement the `Metrics()` call, as that is only used by the Shipper (and its contents would be mostly meaningless in the proxy case since events are not allowed to accumulate).

## Implementation

The proxy queue is loosely based on the implementation of the memory queue, but with many simplifications enabled by its more limited scope. It has three control channels, `getChan`, `pushChan`, and `doneChan`, all unbuffered. Its internal state can only be changed by sending requests to those channels (or closing the channel in the case of `doneChan`), or by closing the done channel on batches it has returned.

### The pipeline

Here is the event data flow through the proxy queue, in the context of the Beats pipeline:

![The proxy queue in context](diagrams/broker.svg)

An input adds an event to the proxy queue by creating a `queue.Producer` via the queue's API and calling its `Publish` function. If the producer was created with an acknowledgment callback, then a pointer to the producer will be included in its event metadata so later stages of the pipeline can notify it when ingestion is complete.

The producer passes an incoming event on to the queue by sending a `pushRequest` to the queue's `pushChan`. The request includes the event, the producer (if acknowledgment is required), a channel on which to receive the response (boolean indicating success or failure), and a flag indicating whether a full queue should block the request until there is room or return immediately with failure. `pushChan` is unbuffered, and any request sent through it is guaranteed to receive a response. If the request's `canBlock` flag is false, that response is guaranteed not to block. If `canBlock` is true, the response is guaranteed to be success unless the queue has been closed.

On the other side of the queue, a worker routine (`queueReader`) requests batches from the queue via its `Get` function, which sends a `getRequest` to the queue's `getChan`. A `getRequest` always blocks until there is data to read or until the queue is closed; as with `pushRequest`, once it is accepted it always returns a response. If the request is successful, the response will be a `proxyqueue.batch` (implementing the `queue.Batch` interface). The `queueReader`'s job is to collect batches from the queue and wrap them in a `publisher.Batch` interface (concrete type `ttlBatch`) that tracks retry metadata used in the final stages of the pipeline.

The wrapped batches generated by the `queueReader` are received by the `eventConsumer`, which is the worker that distributes pipeline batches among the output workers via their shared input channel, and handles retries for output workers that encounter errors.

Only an output worker can complete the life cycle of a batch. In the proxy queue this happens in two stages: when the batch is successfully sent to the Shipper, its `FreeEntries` function is called, which clears the internal reference to the events -- once these are sent, they are no longer needed since they are already enqueued in the Shipper. Then, when the Shipper confirms (via its `PersistedIndex` API, see the Shipper repository for details) that all events from the batch have been processed, the batch's `Done` function is called, which closes the batch's internal channel, `doneChan`.

Finally, the queue's broker routine monitors the `doneChan` of the oldest outstanding batch; when it is closed, the broker invokes the appropriate acknowledgment callbacks and advances to the next oldest batch.

### Acknowledgment tracking

As with other queues, acknowledgments of batches must be globally synchronized by the queue broker, since the pipeline API requires that acknowledgments are sent to producers in the same order the events were generated (out-of-order acknowledgments can cause data loss). The acknowledgments required by any one batch are stored within the batch itself (in the `producerACKs` helper object). The queue broker maintains an ordered linked list of all batches awaiting acknowledgment, and the `select` call in its main loop checks the oldest outstanding batch, calling the appropriate callbacks as it advances.

### The broker loop

All internal control logic is handled in the run loop `broker.run()` in `broker.go`. Its state is stored in these fields:

```go
	queuedEntries      []queueEntry
	blockedRequests    blockedRequests
	outstandingBatches batchList
```

- `queuedEntries` is a list of the events (and producers, if appropriate) currently stored by the queue. Its length is at most `batchSize`.
- `blockedRequests` is a linked list of pending `pushRequest`s from producers that could not be immediately handled because the queue was full. Each one contains a response channel, and the originating producer is listening on that channel waiting for space in the queue. When space is available, events in these requests will be added to `queuedEntries` and the result will be sent to their response channels.
- `outstandingBatches` is a linked list of batches that have been consumed from this queue but not yet acknowledged. It is in the same order as the batches were originally created, so the first entry in the list is always the oldest batch awaiting acknowledgment.

The core loop calls `select` across up to four channels:

- `putChan` accepts requests to add entries to the queue. If the queue is already full (`len(queuedEntries) == batchSize`), the request is either added to `blockedRequests` or returns with immediate failure (depending on the value of `canBlock`). Otherwise, the new entry is added to `queuedEntries` to be included in the next batch.
- `getChan` is enabled only if `queuedEntries` isn't empty (otherwise there would be nothing to return). In that case, a new batch is created with the contents of `queuedEntries`, and metadata for any required future acknowledgments is computed (so that acknowledgment data can persist after the events themselves are freed).
- `outstandingBatches.nextDoneChan()` returns the acknowledgment channel for the oldest outstanding batch; if a read on this channel goes through, it means the channel was closed and the batch has been acknowledged, so the producer and pipeline callbacks are invoked and we advance to the next outstanding batch.
- `doneChan` indicates closure of the queue. In this case we reject any remaining requests in `blockedRequests` and return. (We do not do anything with `outstandingBatches`, since batches that are still unacknowledged at this point should be considered dropped, so we do not want producers to believe they have sent successfully.)

## Possible improvements

The proxy queue is designed to minimize memory use while respecting the established API for the Beats pipeline. However, its inability to buffer incoming events means that raw latency may increase in some scenarios. If benchmarks show that the proxy queue is a CPU or latency bottleneck, there are some natural improvements that would likely yield significant improvements:

- The proxy queue currently buffers at most one batch at a time. Buffering a small constant number of batches instead would potentially block the inputs less often, leading to steadier throughput.
- Unlike the other queues, the proxy queue handles acknowledgments on its main work loop. This may increase latency of control signals if it is given acknowledgment callbacks that perform significant work. In that case, we could add a standalone acknowledgment routine similar to the other queues, so slow acknowledgments do not delay the core control logic.