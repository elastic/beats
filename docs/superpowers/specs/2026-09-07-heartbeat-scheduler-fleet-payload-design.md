# Heartbeat Scheduler Fleet Payload Design

## Goal

Expose aggregate Heartbeat scheduler pressure to Fleet-managed consumers through
the existing Elastic Agent unit-state payload. This allows Kibana's scalable
private-location rebalance task to read pressure from the Fleet agent records it
already loads, without querying Synthetics events or monitoring data streams.

Related issues:

- https://github.com/elastic/beats/issues/53074
- https://github.com/elastic/kibana/issues/281838

## Scope

The first change reports scheduler pressure only. It does not report monitor
identifiers, loaded-monitor hashes, host resources, or make Kibana placement
changes.

Heartbeat reports one bounded entry per configured monitor type:

- `limit`: effective per-type concurrency limit; `0` means unlimited.
- `running`: jobs that hold the per-type semaphore.
- `waiting`: jobs blocked while acquiring the per-type semaphore.
- `runs`: cumulative jobs whose first task started.
- `schedule_delay.count`: cumulative jobs whose first task started.
- `schedule_delay.total_ms`: cumulative delay between scheduled and actual start.
- `schedule_delay.max_ms`: maximum observed delay since process start.

The payload is rooted at `heartbeat.scheduler.jobs`. It contains no monitor IDs
or user-controlled labels.

## Data Flow

1. The scheduler initializes per-type statistics from the effective
   `heartbeat.jobs` configuration after environment overrides are applied.
2. A scheduled job increments `waiting` before acquiring its type semaphore.
3. After acquisition, it decrements `waiting`, increments `running`, and records
   actual first-task start time.
4. Completion decrements `running`.
5. The scheduler records the non-negative difference between scheduled time and
   actual first-task start as schedule delay.
6. Managed Heartbeat snapshots the aggregate values every 30 seconds and calls
   `beat.Manager.SetPayload`.
7. The existing Elastic Agent V2 control protocol attaches the payload to unit
   state. Elastic Agent sends it in Fleet check-ins, and Fleet persists it under
   `agent.components[].units[].payload`.
8. Kibana reads the payload from the `listAgents` response already used by the
   private-location rebalance task.

## Payload

```json
{
  "heartbeat": {
    "scheduler": {
      "jobs": {
        "browser": {
          "limit": 2,
          "running": 2,
          "waiting": 5,
          "runs": 120,
          "schedule_delay": {
            "count": 120,
            "total_ms": 8400,
            "max_ms": 2100
          }
        }
      }
    }
  }
}
```

Unknown or dynamically registered job types are initialized when first used.
Types with no configured limit report `limit: 0` and never wait on the type
semaphore.

## Management Payload Propagation

`BeatV2Manager.SetPayload` already accepts arbitrary structured metadata, but
`agentUnit.UpdateState` suppresses updates whenever status and message are
unchanged. The unit must include payload equality in that suppression decision.

The unit stores a defensive clone of the last payload sent. A payload-only
change triggers `clientUnit.UpdateState`; an identical payload remains
suppressed. Stream-status payload enrichment remains unchanged.

The scheduler reporter publishes only when management is enabled and stops with
Heartbeat. Standalone Heartbeat behavior is unchanged.

## Error Handling

- A canceled semaphore acquisition does not increment `running` or run counters.
- Waiting is decremented after every acquisition attempt.
- Schedule delay is clamped to zero to avoid negative values from clock
  adjustments or immediate schedules.
- Payload delivery errors remain handled by the existing management layer and
  do not affect monitor execution.
- The reporter sends an immediate snapshot after management starts, followed by
  30-second snapshots.

## Testing

Targeted tests will verify:

- Effective configured limits appear in the scheduler snapshot.
- A job blocked on a type semaphore increments `waiting`.
- A job holding the semaphore increments `running`.
- Cancellation leaves waiting and running gauges balanced.
- Schedule-delay counters use actual task start time.
- Heartbeat builds the documented bounded payload.
- A payload-only change reaches `clientUnit.UpdateState`.
- An identical status, message, and payload remains suppressed.

Tests use real scheduler synchronization rather than sleeps where possible.
Existing Heartbeat scheduler and management tests must remain green.

## Consumer Contract

Kibana should select the Heartbeat component's input unit and read
`payload.heartbeat.scheduler.jobs`. Missing payloads mean telemetry is
unavailable and must not make an agent unhealthy or trigger eviction. Placement
should react only to sustained pressure and retain its existing check-in
fallback for older Heartbeat versions.
