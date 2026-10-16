# OpenTelemetry Collector Components in Beats

This is the home of OpenTelemetry Collector components like receivers, processors, exporters, extensions, connectors, etc. that are related to Beats.

The intended structure of this directory is to mimic the structure in the [OpenTelemetry Collector Contrib] repository, specifically:

- Put X receiver in `receiver/xreceiver` subdirectory,
- Put Y processor in `processor/yprocessor` subdirectory,
- Put Z exporter in `exporter/zexporter` subdirectory,
- and so on.

There should be no need to put any Go files directly in this directory.

## Beat receiver memory queues

Beat receivers using `queue.mem` share slab storage while each receiver retains
its configured `queue.mem.events` capacity. A receiver with a stalled output
cannot consume another receiver's event budget. Queue limits include both
queued events and batches awaiting output acknowledgment.

The process-wide capacity is the **sum** of connected receiver limits, rather
than the largest limit. Four receivers using the default 3200-event limit can
retain 12800 events in total. Plan the collector memory budget against that
aggregate capacity and the sizes of the events these receivers produce.
Lowering a receiver limit or shutting it down preserves capacity for its
existing backlog until those events are released; slab storage shrinks lazily
without discarding live events. A receiver configured with `queue.disk` owns
its queue independently.

See the [slab queue capacity notes](../../libbeat/publisher/queue/slabqueue/README.md)
for the allocation model, measurements, and generic uncapped queue behavior.
