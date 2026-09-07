# Heartbeat Scheduler Fleet Payload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish aggregate per-type Heartbeat scheduler pressure in Elastic Agent Fleet check-in unit payloads.

**Architecture:** Heartbeat owns atomic per-job-type scheduler counters and exposes an immutable status snapshot. A managed-mode reporter periodically passes that snapshot to the existing `management.Manager.SetPayload` API. The shared V2 management layer becomes payload-aware so changing telemetry is forwarded even when unit health and message remain unchanged.

**Tech Stack:** Go, Heartbeat scheduler, libbeat Elastic Agent V2 management protocol, `testify`.

## Global Constraints

- Payload root is `heartbeat.scheduler.jobs`.
- Payload contains aggregate monitor types only; no monitor IDs or user-controlled labels.
- `limit: 0` means no per-type concurrency limit.
- Reporter interval is 30 seconds and the first payload is sent immediately.
- Missing telemetry must not change unit health.
- Use test-first red-green cycles for every behavior change.
- Run tests only for the affected packages.

---

### Task 1: Forward Payload-Only Unit Updates

**Files:**
- Modify: `x-pack/libbeat/management/unit.go`
- Test: `x-pack/libbeat/management/unit_test.go`

**Interfaces:**
- Consumes: `agentUnit.UpdateState(state status.Status, msg string, payload map[string]any) error`
- Produces: payload-aware duplicate suppression and a defensive last-payload snapshot.

- [ ] **Step 1: Extend the mock to capture calls and payloads**

Add fields to `mockClientUnit`:

```go
reportedPayload map[string]any
updateCalls     int
```

Update its `UpdateState` implementation:

```go
func (u *mockClientUnit) UpdateState(
	state client.UnitState,
	msg string,
	payload map[string]any,
) error {
	u.reportedState = state
	u.reportedMsg = msg
	u.reportedPayload = payload
	u.updateCalls++
	return nil
}
```

- [ ] **Step 2: Add failing payload-only update tests**

Add focused tests that:

1. Call `UpdateState(status.Running, "Healthy", payloadA)`.
2. Call it again with the same status/message and a different `payloadB`.
3. Assert two client updates occurred and `payloadB` was forwarded.
4. Call it a third time with an equal, independently allocated `payloadB`.
5. Assert no third update occurred.
6. Mutate the caller's original nested map after the first call and verify the
   stored comparison snapshot was not mutated.

- [ ] **Step 3: Run the management test and verify RED**

Run:

```bash
go test ./x-pack/libbeat/management -run 'TestUnitUpdatePayload' -count=1
```

Expected: FAIL because the second call is suppressed solely on equal status and
message.

- [ ] **Step 4: Make duplicate suppression payload-aware**

In `agentUnit`, store the last input-level payload:

```go
inputLevelPayload map[string]any
```

Use `reflect.DeepEqual` for equality and `mapstr.M(payload).Clone()` for a
defensive nested clone. Suppress only when state, message, and payload are all
equal. Save the clone only after deciding to forward the update. Preserve
existing stream payload enrichment.

- [ ] **Step 5: Run management tests and verify GREEN**

Run:

```bash
go test ./x-pack/libbeat/management -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add x-pack/libbeat/management/unit.go x-pack/libbeat/management/unit_test.go
git commit -m "Forward payload-only Beat unit updates" \
  -m "Dynamic managed-mode telemetry changes while health remains stable, so include payload equality in duplicate suppression and retain an immutable comparison snapshot." \
  -m "Assisted-By: Cursor"
```

---

### Task 2: Track Per-Type Scheduler Pressure

**Files:**
- Create: `heartbeat/scheduler/status.go`
- Create: `heartbeat/scheduler/status_test.go`
- Modify: `heartbeat/scheduler/scheduler.go`
- Modify: `heartbeat/scheduler/schedjob.go`
- Test: `heartbeat/scheduler/schedjob_test.go`

**Interfaces:**
- Produces:

```go
type ScheduleDelayStatus struct {
	Count   uint64 `json:"count"`
	TotalMS uint64 `json:"total_ms"`
	MaxMS   uint64 `json:"max_ms"`
}

type JobTypeStatus struct {
	Limit         int64               `json:"limit"`
	Running       int64               `json:"running"`
	Waiting       int64               `json:"waiting"`
	Runs          uint64              `json:"runs"`
	ScheduleDelay ScheduleDelayStatus `json:"schedule_delay"`
}

type Status struct {
	Jobs map[string]JobTypeStatus `json:"jobs"`
}

func (s *Scheduler) Status() Status
```

- [ ] **Step 1: Write failing status snapshot tests**

Construct a scheduler with browser limit 2 and assert:

```go
status := scheduler.Status()
assert.Equal(t, int64(2), status.Jobs["browser"].Limit,
	"configured browser limit should be reported")
assert.Equal(t, int64(0), status.Jobs["browser"].Running,
	"new scheduler should report no running browser jobs")
assert.Equal(t, int64(0), status.Jobs["browser"].Waiting,
	"new scheduler should report no waiting browser jobs")
```

Also request a previously unknown `http` job type through the scheduler's
internal accessor and assert it reports limit 0.

- [ ] **Step 2: Run the status test and verify RED**

Run:

```bash
go test ./heartbeat/scheduler -run 'TestSchedulerStatus' -count=1
```

Expected: FAIL because `Scheduler.Status` does not exist.

- [ ] **Step 3: Implement atomic per-type status storage**

In `status.go`, define an internal entry:

```go
type jobTypeStats struct {
	limit        int64
	running      atomic.Int64
	waiting      atomic.Int64
	runs         atomic.Uint64
	delayCount   atomic.Uint64
	delayTotalMS atomic.Uint64
	delayMaxMS   atomic.Uint64
}
```

Add a mutex-protected `map[string]*jobTypeStats` to `Scheduler`. Initialize
configured entries in `Create`, and lazily initialize unknown job types with
limit 0. `Status` must copy values into a new map without exposing atomics.

- [ ] **Step 4: Run the status test and verify GREEN**

Run:

```bash
go test ./heartbeat/scheduler -run 'TestSchedulerStatus' -count=1
```

Expected: PASS.

- [ ] **Step 5: Write a failing semaphore-accounting test**

Use a browser limit of 1. Start one blocking job and wait for it to signal that
its task is executing. Start a second job and use `require.Eventually` only to
observe the deterministic state transition:

```go
assert.Equal(t, int64(1), status.Jobs["browser"].Running,
	"one browser job should hold the type semaphore")
assert.Equal(t, int64(1), status.Jobs["browser"].Waiting,
	"the second browser job should wait for the type semaphore")
```

Release the first job and verify both gauges return to zero after both jobs
complete. Add a canceled-context case and verify no running/runs increment.

- [ ] **Step 6: Run the accounting test and verify RED**

Run:

```bash
go test ./heartbeat/scheduler -run 'TestJobTypePressure' -count=1
```

Expected: FAIL because type semaphore acquisition is not instrumented.

- [ ] **Step 7: Instrument the type semaphore**

Store a `*jobTypeStats` on each `schedJob`. In `run`:

1. Increment `waiting` only when a type semaphore exists.
2. Attempt acquisition.
3. Decrement `waiting` after every attempt.
4. Return without incrementing `running` when acquisition fails.
5. Increment `running` after successful acquisition or immediately for an
   unlimited type.
6. Decrement `running` after the full recursive job finishes.
7. Increment `runs` only when the first task actually starts.

- [ ] **Step 8: Run scheduler tests and verify GREEN**

Run:

```bash
go test ./heartbeat/scheduler -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add heartbeat/scheduler
git commit -m "Track Heartbeat scheduler pressure by monitor type" \
  -m "Expose the effective per-type limit and jobs running or waiting on its semaphore so Fleet consumers can distinguish actual Synthetics saturation from global scheduler activity." \
  -m "Assisted-By: Cursor"
```

---

### Task 3: Record Actual Schedule Delay

**Files:**
- Modify: `heartbeat/scheduler/scheduler.go`
- Modify: `heartbeat/scheduler/status.go`
- Test: `heartbeat/scheduler/status_test.go`
- Test: `heartbeat/scheduler/scheduler_test.go`

**Interfaces:**
- Consumes: the scheduled timestamp captured by `Scheduler.runTaskOnce`.
- Produces: cumulative count, total milliseconds, and maximum milliseconds in
  `JobTypeStatus.ScheduleDelay`.

- [ ] **Step 1: Write a failing delay-aggregation test**

Add an internal `recordDelay(time.Duration)` test using delays of 100ms, 900ms,
and -1s. Assert:

```go
assert.Equal(t, uint64(3), delay.Count,
	"all started jobs should contribute a delay sample")
assert.Equal(t, uint64(1000), delay.TotalMS,
	"negative delay should be clamped to zero")
assert.Equal(t, uint64(900), delay.MaxMS,
	"maximum delay should retain the largest sample")
```

- [ ] **Step 2: Run and verify RED**

Run:

```bash
go test ./heartbeat/scheduler -run 'TestJobTypeScheduleDelay' -count=1
```

Expected: FAIL because delay aggregation is absent.

- [ ] **Step 3: Implement lock-free delay aggregation**

Increment count and total with atomics. Update max with a compare-and-swap loop.
Clamp negative durations to zero before conversion to milliseconds.

- [ ] **Step 4: Run and verify GREEN**

Run:

```bash
go test ./heartbeat/scheduler -run 'TestJobTypeScheduleDelay' -count=1
```

Expected: PASS.

- [ ] **Step 5: Write a failing scheduled-versus-started integration test**

Add a scheduler test with a type limit of 1:

1. Hold the first job.
2. Schedule a second job for a known timestamp.
3. Keep it waiting longer than the scheduled timestamp.
4. Release the first job.
5. Assert the second job's recorded delay includes its semaphore wait.

Use channels to establish ordering; use a broad lower bound only for the elapsed
duration assertion.

- [ ] **Step 6: Run and verify RED**

Run:

```bash
go test ./heartbeat/scheduler -run 'TestScheduleDelayIncludesTypeLimitWait' -count=1
```

Expected: FAIL because scheduled timestamps are not connected to actual first
task start.

- [ ] **Step 7: Capture scheduled timestamps**

Introduce a private scheduled callback that receives both scheduled and timer
trigger timestamps. `runTaskOnce` captures `runAt` in its wrapper. After
`schedJob.run` returns an actual started timestamp, record
`startedAt.Sub(scheduledAt)` on that job type. Do not record canceled jobs or
maintenance-window skips.

- [ ] **Step 8: Run all scheduler tests**

Run:

```bash
go test ./heartbeat/scheduler -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add heartbeat/scheduler
git commit -m "Measure Heartbeat schedule delay by monitor type" \
  -m "Record actual first-task start relative to its due time, including time queued behind the per-type concurrency limit, so consumers can detect sustained lateness." \
  -m "Assisted-By: Cursor"
```

---

### Task 4: Publish Scheduler Status Through Managed Unit Payloads

**Files:**
- Create: `heartbeat/beater/scheduler_payload.go`
- Create: `heartbeat/beater/scheduler_payload_test.go`
- Modify: `heartbeat/beater/heartbeat.go`

**Interfaces:**
- Consumes: `scheduler.Status()` and an interface containing
  `SetPayload(map[string]any)`.
- Produces: immediate and 30-second managed-mode unit payload updates.

- [ ] **Step 1: Write a failing payload-shape test**

Define a narrow reporter dependency:

```go
type payloadSetter interface {
	SetPayload(map[string]any)
}
```

Test a pure payload builder against:

```go
map[string]any{
	"heartbeat": map[string]any{
		"scheduler": map[string]any{
			"jobs": map[string]any{
				"browser": map[string]any{
					"limit":   int64(2),
					"running": int64(0),
					"waiting": int64(0),
					"runs":    uint64(0),
					"schedule_delay": map[string]any{
						"count":    uint64(0),
						"total_ms": uint64(0),
						"max_ms":   uint64(0),
					},
				},
			},
		},
	},
}
```

Build plain JSON-compatible maps rather than placing `scheduler.Status` structs
in the payload because `structpb.NewStruct` does not accept arbitrary Go
structs.

- [ ] **Step 2: Run and verify RED**

Run:

```bash
go test ./heartbeat/beater -run 'TestSchedulerPayload' -count=1
```

Expected: FAIL because the builder does not exist.

- [ ] **Step 3: Implement the pure payload builder**

Keep it unexported and return a fresh map around the immutable scheduler status.

- [ ] **Step 4: Run and verify GREEN**

Run:

```bash
go test ./heartbeat/beater -run 'TestSchedulerPayload' -count=1
```

Expected: PASS.

- [ ] **Step 5: Write failing reporter lifecycle tests**

Inject a ticker channel instead of sleeping. Verify:

- one payload is sent immediately;
- one payload is sent for each tick;
- cancellation stops further sends;
- the reporter is not started when management is disabled.

- [ ] **Step 6: Run and verify RED**

Run:

```bash
go test ./heartbeat/beater -run 'TestSchedulerPayloadReporter' -count=1
```

Expected: FAIL because no reporter exists.

- [ ] **Step 7: Implement and wire the reporter**

After `b.Manager.Start()` succeeds in `Heartbeat.Run`, start the reporter only
when `b.Manager.Enabled()` is true. Send immediately, then every 30 seconds.
Stop it before `b.Manager.Stop()` via deferred cancellation. Reporter failures
cannot affect monitor execution because `SetPayload` has no return value.

- [ ] **Step 8: Run affected package tests**

Run:

```bash
go test ./heartbeat/beater ./heartbeat/scheduler ./x-pack/libbeat/management -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add heartbeat/beater
git commit -m "Publish Heartbeat scheduler pressure to Fleet" \
  -m "Send bounded scheduler snapshots through managed unit payloads so Fleet-backed tasks can consume pressure using their existing agent check-in reads." \
  -m "Assisted-By: Cursor"
```

---

### Task 5: Changelog and Final Verification

**Files:**
- Create: the timestamped YAML file printed by
  `elastic-agent-changelog-tool new --component heartbeat --kind enhancement`

**Interfaces:**
- Consumes: completed implementation.
- Produces: release-note metadata and verified draft-PR branch.

- [ ] **Step 1: Format changed Go files**

Run:

```bash
gofmt -w heartbeat/scheduler/status.go heartbeat/scheduler/status_test.go \
  heartbeat/scheduler/scheduler.go heartbeat/scheduler/schedjob.go \
  heartbeat/scheduler/scheduler_test.go heartbeat/scheduler/schedjob_test.go \
  heartbeat/beater/scheduler_payload.go heartbeat/beater/scheduler_payload_test.go \
  heartbeat/beater/heartbeat.go x-pack/libbeat/management/unit.go \
  x-pack/libbeat/management/unit_test.go
```

- [ ] **Step 2: Create the changelog fragment**

Run:

```bash
go run github.com/elastic/elastic-agent-changelog-tool@latest new \
  --component heartbeat --kind enhancement
```

Set the summary to:

```yaml
summary: Report per-type scheduler pressure through Elastic Agent unit state
```

- [ ] **Step 3: Run fresh targeted verification**

Run:

```bash
go test -race ./heartbeat/scheduler ./heartbeat/beater ./x-pack/libbeat/management -count=1
git diff --check upstream/main...HEAD
```

Expected: all tests PASS; diff check exits 0.

- [ ] **Step 4: Review the complete branch diff**

Run:

```bash
git diff --stat upstream/main...HEAD
git diff upstream/main...HEAD
```

Confirm the branch contains only the spec, plan, management payload fix,
scheduler telemetry, managed reporter, tests, and changelog.

- [ ] **Step 5: Commit final metadata**

```bash
git add docs/superpowers/plans changelog/fragments
git commit -m "Add Heartbeat scheduler telemetry changelog" \
  -m "Document the managed scheduler-pressure payload for release notes and retain the implementation plan used to produce the change." \
  -m "Assisted-By: Cursor"
```

- [ ] **Step 6: Push and open a draft PR**

```bash
git push -u origin heartbeat-scheduler-fleet-payload
gh pr create --repo elastic/beats --draft \
  --title "Report Heartbeat scheduler pressure through Fleet" \
  --body "$(cat <<'EOF'
## Proposed commit message

Report Heartbeat scheduler pressure through Fleet

## Checklist

- [x] My code follows the style guidelines of this project
- [x] I have commented my code, particularly in hard-to-understand areas
- [x] I have made corresponding changes to the documentation
- [x] I have made corresponding change to the default configuration files
- [x] I have added tests that prove my fix is effective or that my feature works

## Author's Checklist

- [ ] Kibana consumption is intentionally left to a follow-up in elastic/kibana#281838.

## How to test this PR locally

Run:

`go test -race ./heartbeat/scheduler ./heartbeat/beater ./x-pack/libbeat/management -count=1`

## Related issues

Closes #53074

Related: elastic/kibana#281838

## Use cases

Fleet-managed consumers can read per-monitor-type Heartbeat scheduler pressure
from `agent.components[].units[].payload` without querying Synthetics documents
or Elastic Agent monitoring data streams.

## Screenshots

Not applicable.

## Logs

Not applicable.
EOF
)"
```

The PR body must:

- summarize per-type pressure, delay, and payload-only propagation;
- link `Closes #53074`;
- link `elastic/kibana#281838`;
- list the exact targeted race-test command;
- note that the Kibana consumer is a follow-up.
