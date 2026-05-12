# Packetbeat Receiver: DirectQueue Analysis

## Why DirectQueue is safe for packetbeat

When running as a beatreceiver in an OTel Collector, packetbeat uses `DirectQueue`
mode which bypasses the publisher queue and sends events synchronously inline.
This document explains why the existing architecture already tolerates this.

## Event flow: sniffer to output

Everything from packet capture through protocol decoding through event publishing
runs in a **single goroutine per network interface**:

```
Sniffer.Run()
  └─ one goroutine per interface (via errgroup)
      └─ sniffHandle() loop:
           for {
               data := handle.ReadPacketData()    // read from kernel
               dec.OnPacket(data)                  // decode packet (same goroutine)
                 └─ protocol plugin (HTTP, DNS, etc.)
                      └─ publishTransaction(event)
                           └─ results(event)       // reporter function
           }
```

The reporter function returned by `TransactionPublisher.CreateReporter()` is a
closure that sends events to a buffered channel (capacity 3):

```go
// publish.go:131-137
return func(event beat.Event) {
    select {
    case ch <- event:     // blocks if channel full — NO default case
    case <-p.done:
    }
}, nil
```

A separate worker goroutine drains this channel and calls `client.Publish()`:

```go
// publish.go:140-154
func (p *TransactionPublisher) worker(ch chan beat.Event, client beat.Client) {
    for {
        select {
        case event := <-ch:
            pub, _ := p.processor.Run(&event)
            client.Publish(*pub)
        case <-p.done:
            return
        }
    }
}
```

## Backpressure is already blocking

The channel send in the reporter has **no `default` case**. When the channel is
full (worker hasn't drained it), the sniffer goroutine blocks. While blocked:

- `ReadPacketData()` is not called
- The kernel's packet capture buffer fills
- Eventually the kernel drops packets

This means the sniffer **already blocks on a slow pipeline today**. The 3-slot
channel just provides 3 events of slack before the block.

## What DirectQueue changes

With DirectQueue, the channel and worker goroutine are bypassed. The reporter
calls `client.Publish()` directly in the sniffer goroutine. If the OTel pipeline
is slow, `Publish` blocks, the sniffer goroutine stalls, and `ReadPacketData()`
isn't called — the same behavior as when the channel is full.

| Aspect | DefaultQueue (with channel) | DirectQueue |
|--------|---------------------------|-------------|
| Sniffer blocks when pipeline slow | Yes (channel full) | Yes (Publish blocks) |
| Buffer before block | 3 events + queue (3200) | None |
| Where drops happen | Queue (`DropIfFull`) | Kernel buffer overflow |
| Goroutines | sniffer + worker per protocol | sniffer only |
| Latency overhead | channel + queue + consumer + worker | direct call |

## DropIfFull behavior

In standalone mode, packetbeat sets `PublishMode: beat.DropIfFull` for live
capture. This makes the worker's `client.Publish()` use `TryPublish` on the
queue — a non-blocking call that drops events when the queue is full. This keeps
the worker goroutine fast so the channel rarely fills.

With DirectQueue there is no queue, so `DropIfFull` has no effect. Backpressure
flows directly from the OTel pipeline to the sniffer goroutine. The OTel
pipeline's own buffering (batch processor, exporter queue) controls throughput.

## Summary

The fundamental backpressure path — slow output stalls the sniffer — is identical
in both modes. DirectQueue removes intermediate buffering (channel + queue) but
does not change the blocking behavior. Packet drops under sustained load happen
at the kernel level regardless of mode; DirectQueue just makes the path shorter.
