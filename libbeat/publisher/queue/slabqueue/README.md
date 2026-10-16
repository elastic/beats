# Shared slab queue capacity

Each in-memory Beat receiver connects a capped queue to a process-wide slab
pool. The pool's steady-state slot target is the sum of receiver caps. A full
receiver can retain its queued and in-flight events while another receiver
continues to admit events up to its own cap. Slots remain live until the output
acknowledges or explicitly releases their batches; reading the FIFO does not
return their capacity.

When a queue's cap is lowered below its retained backlog, its contribution is
`max(cap, live events)`. A closed queue contributes its remaining live events
until they are released. These temporary contributions preserve other
receivers' capacity through resizing and graceful shutdown without dropping
the existing backlog. Physical capacity can exceed the target during lazy
shrink: only free trailing slots and whole trailing chunks are reclaimed.

Generic queues connected without a positive `SetTarget` retain their existing
shared-budget behavior. They reserve no capacity and may use every available
pool slot, so a pool containing uncapped queues does not guarantee capacity
isolation. Pools containing only uncapped queues keep their existing target.
The OTel receiver controller always configures a positive per-receiver cap.

## Memory budget

Changing from the largest receiver cap to their sum raises the event retention
bound. For `N` receivers each configured for `E` events, the steady-state
capacity is `N * E` rather than `E`. A single receiver's capacity is unchanged.

Storage uses 4096-slot chunks. On linux/amd64, one `publisher.Event` slot is
112 bytes, so its slab storage occupies
`ceil(target / 4096) * 4096 * 112` bytes. This excludes the event's referenced
payload, free-list arrays, batches, and queue metadata. In-flight exporter
representation and output buffers can add memory outside the pool.

`BenchmarkCappedPoolCapacity` measures empty pools with 3200 events configured
per receiver. A local Go 1.26.8 linux/amd64 run measured:

| Receivers | Aggregate slots | Slab storage | Total allocated per empty pool | Earlier largest-cap pool |
| --- | ---: | ---: | ---: | ---: |
| 1 | 3200 | 448 KiB | about 514 KiB | about 513 KiB |
| 4 | 12800 | 1792 KiB | about 2105 KiB | about 515 KiB |
| 8 | 25600 | 3136 KiB | about 3824 KiB | about 518 KiB |
| 16 | 51200 | 5824 KiB | about 7287 KiB | about 524 KiB |

The earlier pool figures were measured with the same benchmark using source
from the upstream baseline, whose target remains 3200 slots at every receiver
count.

These are allocation measurements, not peak RSS or a bound on retained event
payloads. For example, four full 3200-event receivers holding an average of
1 KiB of referenced payload each can retain about 12.5 MiB of payload in
addition to the queue storage. The earlier shared-largest-cap design retained
at most about 3.125 MiB of those payloads, at the cost of cross-receiver
starvation. Configure receiver limits so their aggregate capacity and expected
payload fit the collector's memory budget.

Reproduce the allocation measurement with:

```sh
go test -run '^$' -bench '^BenchmarkCappedPoolCapacity$' -benchmem -benchtime=100x ./libbeat/publisher/queue/slabqueue
```

`BenchmarkSlabQueuePool` assigns each receiver a share of a fixed aggregate
4096-event cap when comparing throughput to a standalone shared memqueue.
This holds total capacity constant while exercising capped receiver queues.
