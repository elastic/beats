# Journald all-boots integration test plan

## Objective
Add a new Filebeat integration test that validates journald ingestion from more than one boot across different `journalctl` versions, with special attention to versions `<242` that do not support `--boot all`.

## Target files
- `filebeat/tests/integration/journald_test.go`
- `filebeat/tests/integration/testdata/filebeat_journald_all_boots.yml` (new)

## Test design
### 1) Discover available boots (runtime precondition)
- Execute `journalctl --list-boots --no-pager`.
- Parse non-empty output lines.
- Assert boot count `> 1`; if not, fail test (as requested).
- Parse the two oldest boot entries (first two entries in `--list-boots` output):
  - boot ID (32 hex chars)
  - start timestamp text
  - boot offset (first column from `--list-boots`)
- Keep oldest boot metadata for diagnostics and traceability.

### 2) Compute wait target from the two oldest boots
- Count journal entries for each of the two oldest boots, for example:
  - `journalctl -b <offset> --output=json --no-pager --quiet | wc -l`
- Compute `expectedMessages = oldestCount + secondOldestCount`.
- Use `expectedMessages` as the number of messages to wait for in Filebeat output.

### 3) Start Filebeat with file output
- Use `integration.NewBeat(...)` and write a dedicated config file for this test.
- Start Filebeat and wait for `journalctl started` log line.
- Wait until output has at least `expectedMessages` events using `assert.EventuallyWithT` over `CountFileLines(...)`.
  - Do not use `WaitPublishedEvents` here, because it expects an exact count and this input is continuous (`--follow`).

### 4) Read published events and assert multi-boot coverage
- Read events from file output with `integration.GetEventsFromFileOutput[...]`.
- Extract `journald.host.boot_id` from each event.
- Build a set of distinct boot IDs.
- Assert:
  - the set has at least 2 distinct boot IDs

### 5) Keep failure diagnostics useful
- On failure, log:
  - `journalctl --list-boots` raw output
  - parsed oldest and second-oldest boot IDs
  - per-boot entry counts and computed `expectedMessages`
  - number of events read
  - first few distinct boot IDs found in output

## Filebeat configuration to use (highlighted)
**Use an unfiltered journald input with default input settings and file output.**  
Defaults are sufficient for this test; no explicit `enabled` or `seek` is required.

```yaml
filebeat.inputs:
  - type: journald
    id: journald-all-boots

path.home: %s

queue.mem:
  flush.timeout: 0

output:
  file:
    path: ${path.home}
    filename: "output"
    rotate_on_startup: false

logging:
  level: debug
  selectors:
    - "input.journald"
    - "input.journald.reader"
    - "input.journald.reader.journalctl-runner"
```

## Proposed test function and helpers
- New test:
  - `TestJournaldInputReadsMessagesFromAllBoots`
- Helper functions in `journald_test.go`:
  - `listBoots(t *testing.T) (boots []bootInfo, raw string)`
  - `countBootEntries(t *testing.T, bootOffset string) int`
  - `waitForAtLeastPublishedEvents(t *testing.T, b *integration.BeatProc, min int, timeout time.Duration)`
  - `distinctBootIDs(events []eventType) map[string]struct{}`

## Validation plan
Run this test in each prepared VM:
- `239`, `240`, `241`, `242`, `250`

Command:
- `go test -tags integration ./filebeat/tests/integration -run TestJournaldInputReadsMessagesFromAllBoots -v`

Expected:
- Passes when VM has more than one boot in journald.
- Fails early with explicit message when only one boot exists.

## Unresolved questions
- Should we cap `expectedMessages` if the two oldest boots have very large journals, to avoid long-running CI?
