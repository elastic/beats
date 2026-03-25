# Remove azure-eventhub processor v1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the deprecated processor v1 from the azure-eventhub input, keeping v2 as the only implementation, with a graceful deprecation warning for users who still configure v1.

**Architecture:** Delete v1 implementation files, update routing to always use v2 (with warning on v1 config), simplify config validation to remove v1 branches, rewrite tests that depended on v1 types, update documentation, and clean up deprecated SDK dependencies.

**Tech Stack:** Go, Azure SDK for Go (azeventhubs), Beats input v2 API

**Spec:** `docs/superpowers/specs/2026-03-25-remove-azureeventhub-processor-v1-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `x-pack/filebeat/input/azureeventhub/v1_input.go` | Delete | V1 processor implementation |
| `x-pack/filebeat/input/azureeventhub/v1_input_test.go` | Delete | V1 processor tests |
| `x-pack/filebeat/input/azureeventhub/file_persister_test.go` | Delete | V1-only SDK test |
| `x-pack/filebeat/input/azureeventhub/tracer.go` | Delete | Legacy SDK tracing (devigned/tab) |
| `x-pack/filebeat/input/azureeventhub/azureeventhub_integration_test.go` | Delete | Stale v1 integration test |
| `x-pack/filebeat/input/azureeventhub/input.go` | Modify | Remove v1 routing, dead code, legacy imports; add warning |
| `x-pack/filebeat/input/azureeventhub/config.go` | Modify | Remove v1 validation branches |
| `x-pack/filebeat/input/azureeventhub/config_test.go` | Modify | Remove v1 test cases, update v1 config tests |
| `x-pack/filebeat/input/azureeventhub/input_test.go` | Modify | Remove v1 tests, deprecated imports; add fallback test |
| `x-pack/filebeat/input/azureeventhub/metrics_test.go` | Modify | Rewrite to use messageDecoder directly instead of eventHubInputV1 |
| `x-pack/filebeat/input/azureeventhub/client_secret.go` | Modify | Remove v1 comment |
| `x-pack/filebeat/input/azureeventhub/README.md` | Modify | Update v1 references |
| `docs/reference/filebeat/filebeat-input-azure-eventhub.md` | Modify | Remove v1 example, update descriptions |
| `go.mod` / `go.sum` | Modify | go mod tidy |

---

### Task 1: Delete v1 implementation files

**Files:**
- Delete: `x-pack/filebeat/input/azureeventhub/v1_input.go`
- Delete: `x-pack/filebeat/input/azureeventhub/v1_input_test.go`
- Delete: `x-pack/filebeat/input/azureeventhub/file_persister_test.go`
- Delete: `x-pack/filebeat/input/azureeventhub/tracer.go`
- Delete: `x-pack/filebeat/input/azureeventhub/azureeventhub_integration_test.go`

- [ ] **Step 1: Delete the v1 implementation and related files**

```bash
cd x-pack/filebeat/input/azureeventhub
rm v1_input.go v1_input_test.go file_persister_test.go tracer.go azureeventhub_integration_test.go
```

- [ ] **Step 2: Commit the deletions**

```bash
git add -u x-pack/filebeat/input/azureeventhub/
git commit -m "azureeventhub: delete processor v1 implementation and related files

Remove v1_input.go, v1_input_test.go, file_persister_test.go,
tracer.go, and azureeventhub_integration_test.go.

These files implement the deprecated processor v1 using the
azure-event-hubs-go/v3 SDK which Microsoft no longer supports."
```

**Note:** The build will be broken after this commit until subsequent tasks clean up references to deleted types/functions. This is expected.

---

### Task 2: Update input.go — remove v1 routing and dead code, add warning

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/input.go`

- [ ] **Step 1: Remove legacy imports and dead code from input.go**

Remove these imports:
```go
"github.com/Azure/go-autorest/autorest/azure"
"github.com/devigned/tab"
```

Remove the `environments` map variable (lines 31-36):
```go
var environments = map[string]azure.Environment{
	azure.ChinaCloud.ResourceManagerEndpoint:        azure.ChinaCloud,
	azure.GermanCloud.ResourceManagerEndpoint:       azure.GermanCloud,
	azure.PublicCloud.ResourceManagerEndpoint:       azure.PublicCloud,
	azure.USGovernmentCloud.ResourceManagerEndpoint: azure.USGovernmentCloud,
}
```

- [ ] **Step 2: Remove tab.Register call and tracer reference**

Remove the tracing block in `Create()` (lines 76-80):
```go
// Register the logs tracer only if the environment variable is
// set to avoid the overhead of the tracer in environments where
// it's not needed.
if os.Getenv("BEATS_AZURE_EVENTHUB_INPUT_TRACING_ENABLED") == "true" {
	tab.Register(&logsOnlyTracer{logger: m.log})
}
```

Also remove the `"os"` import if it becomes unused after this change.

- [ ] **Step 3: Replace the processor version switch with v1 warning + always v2**

Replace the `switch config.ProcessorVersion` block (lines 89-96) with:

```go
if config.ProcessorVersion == processorV1 {
	m.log.Warn("processor v1 is no longer available, using v2. The processor_version option will be removed in a future release.")
}

return newEventHubInputV2(config, m.log)
```

- [ ] **Step 4: Add FIPS TODO comment**

Update the `ExcludeFromFIPS` comment to add a TODO:

```go
// ExcludeFromFIPS = true to prevent this input from being used in FIPS-capable
// Filebeat distributions. This input indirectly uses algorithms that are not
// FIPS-compliant. Specifically, the input depends on the
// github.com/Azure/azure-sdk-for-go/sdk/azidentity package which, in turn,
// depends on the golang.org/x/crypto/pkcs12 package, which is not FIPS-compliant.
//
// TODO: investigate whether FIPS exclusion is still needed now that
// the deprecated azure-event-hubs-go SDK has been removed.
ExcludeFromFIPS: true,
```

- [ ] **Step 5: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/input.go
git commit -m "azureeventhub: update input.go to remove v1 routing and add deprecation warning

- Remove go-autorest and devigned/tab imports
- Remove environments map (only used by v1)
- Remove tab.Register tracing call (v1-only)
- Replace processor version switch with warning + always v2
- Add TODO for FIPS exclusion investigation"
```

---

### Task 3: Simplify config validation — remove v1 branches

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/config.go`

- [ ] **Step 1: Simplify validateStorageAccountAuthForConnectionString()**

Replace the function body (lines 278-290) to remove the v1 branch. Since v2 doesn't validate storage account auth at this point (it's validated later in `validateStorageAccountConfigV2`), the function can simply return nil:

```go
func (conf *azureInputConfig) validateStorageAccountAuthForConnectionString() error {
	// Storage account validation for connection_string auth is handled
	// by validateStorageAccountConfigV2().
	return nil
}
```

- [ ] **Step 2: Simplify validateStorageAccountAuthForClientSecret()**

Replace the function body (lines 313-326) to remove the v1 branch:

```go
func (conf *azureInputConfig) validateStorageAccountAuthForClientSecret() error {
	// For connection_string auth type with processor v2: Storage Account uses
	// the same client_secret credentials as Event Hub.
	// The client_secret credentials are already validated above for Event Hub.
	return nil
}
```

- [ ] **Step 3: Simplify validateStorageAccountConfig()**

Replace the function (lines 407-424) to remove the v1 branch and the switch:

```go
func (conf *azureInputConfig) validateStorageAccountConfig(logger *logp.Logger) error {
	return conf.validateStorageAccountConfigV2(logger)
}
```

- [ ] **Step 4: Simplify checkUnsupportedParams()**

In `checkUnsupportedParams()` (lines 491-507), the `SAKey` deprecation warning is guarded by `conf.ProcessorVersion == processorV2`. Since v2 is now the only processor, remove the guard:

Change:
```go
if conf.ProcessorVersion == processorV2 {
	if conf.SAKey != "" {
		logger.Warnf("storage_account_key is not used in processor v2, please remove it from the configuration (config: storage_account_key)")
	}
}
```

To:
```go
if conf.SAKey != "" {
	logger.Warnf("storage_account_key is deprecated, please use storage_account_connection_string instead (config: storage_account_key)")
}
```

- [ ] **Step 5: Update config comments**

Update the `SAKey` field comment (line 32) from:
```go
// SAKey is used to connect to the storage account (processor v1 only)
```
to:
```go
// SAKey is the storage account key. Deprecated: use SAConnectionString instead.
```

Update the `SAConnectionString` field comment (line 34) from:
```go
// SAConnectionString is used to connect to the storage account (processor v2 only)
```
to:
```go
// SAConnectionString is used to connect to the storage account.
```

Update the `ProcessorVersion` field comment (line 111-112) from:
```go
// ProcessorVersion controls the processor version to use.
// Possible values are v1 and v2 (processor v2 only). The default is v2.
```
to:
```go
// ProcessorVersion controls the processor version to use. The default is v2.
// Note: v1 is no longer available. If set to "v1", the input will log a warning
// and use v2. This option will be removed in a future release.
```

- [ ] **Step 5: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/config.go
git commit -m "azureeventhub: simplify config validation by removing v1 branches

Remove processor v1 validation paths from:
- validateStorageAccountAuthForConnectionString
- validateStorageAccountAuthForClientSecret
- validateStorageAccountConfig

Update field comments to reflect v1 removal."
```

---

### Task 4: Update config_test.go — remove v1 test cases

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/config_test.go`

- [ ] **Step 1: Update TestValidate to use v2**

In `TestValidate` (line 38), change the test config from `ProcessorVersion: "v1"` to use v2 config (add `SAConnectionString` instead of `SAKey`):

```go
t.Run("Sanitize storage account containers with underscores", func(t *testing.T) {
	config := defaultConfig()
	config.ConnectionString = "Endpoint=sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SECRET"
	config.EventHubName = "event_hub_00"
	config.SAName = "teststorageaccount"
	config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=secret;EndpointSuffix=core.windows.net"
	config.SAContainer = "filebeat-activitylogs-event_hub_00"

	require.NoError(t, config.Validate())

	assert.Equal(
		t,
		"filebeat-activitylogs-event-hub-00",
		config.SAContainer,
		"underscores (_) not replaced with hyphens (-)",
	)
})
```

- [ ] **Step 2: Convert TestValidateConnectionStringV1 to use v2 configs**

The tests in `TestValidateConnectionStringV1` (lines 59-107) are testing connection string validation (entity path matching), which still applies to v2. Convert them to use v2 config by replacing `ProcessorVersion: "v1"` and `SAKey` with `ProcessorVersion: "v2"` and `SAConnectionString`. Rename the test function to `TestValidateConnectionString` since there's no v1/v2 distinction anymore.

For the three sub-tests, change:
- `config.ProcessorVersion = "v1"` → remove (default is v2)
- `config.SAKey = "my-secret"` → `config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-secret;EndpointSuffix=core.windows.net"`

- [ ] **Step 3: Remove v1-specific test cases from TestClientSecretConfigValidation**

Remove these two test cases (lines 264-297):
- `"valid client_secret config with processor v1"` — tests v1 with SAKey
- `"client_secret config with processor v1 missing storage account key"` — tests v1 error for missing SAKey

- [ ] **Step 4: Remove v1-specific test cases from TestConnectionStringConfigValidation**

Remove these two test cases:
- `"valid connection_string config with processor v1"` (lines 371-384) — tests v1 with SAKey
- `"connection_string config with processor v1 missing storage account key"` (lines 399-412) — tests v1 error for missing SAKey

- [ ] **Step 5: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/config_test.go
git commit -m "azureeventhub: remove v1-specific config test cases

Convert v1 connection string tests to v2 config.
Remove v1-specific client_secret and connection_string test cases."
```

---

### Task 5: Update input_test.go — remove v1 tests, add fallback test

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/input_test.go`

- [ ] **Step 1: Remove deprecated SDK imports**

Remove these imports from input_test.go:
```go
"github.com/Azure/go-autorest/autorest/azure"
eventhub "github.com/Azure/azure-event-hubs-go/v3"
```

- [ ] **Step 2: Remove TestGetAzureEnvironment (lines 35-51)**

Delete the entire `TestGetAzureEnvironment` function — it tests `getAzureEnvironment()` which was defined in the deleted `v1_input.go`.

- [ ] **Step 3: Remove TestProcessEvents (lines 53-98)**

Delete the entire `TestProcessEvents` function — it creates `eventHubInputV1{}` which no longer exists.

- [ ] **Step 4: Remove the commented-out TestNewInputDone (lines 100-109)**

Delete the commented-out code block.

- [ ] **Step 5: Remove the defaultTestConfig variable (lines 27-33)**

Delete `defaultTestConfig` — it was only used by `TestProcessEvents` in this file and `metrics_test.go` (which will be rewritten in the next task).

- [ ] **Step 6: Clean up unused imports**

After removing the tests and variable, clean up imports. The remaining code (`fakeClient` struct and its methods) needs only:
```go
import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
)
```

Remove: `"fmt"`, `"testing"`, `"time"`, `"github.com/elastic/elastic-agent-libs/logp"`, `"github.com/elastic/elastic-agent-libs/monitoring"`, `"github.com/stretchr/testify/assert"`.

**Note:** Keep the `fakeClient` type and its methods — they're used by `metrics_test.go`.

- [ ] **Step 7: Add fallback test for processor_version v1**

Add a test that verifies the v1 → v2 fallback behavior in `Create()`. This test should confirm that when `processor_version: "v1"` is configured, the input manager logs a warning and still creates a v2 input without error.

```go
func TestCreateWithProcessorV1FallsBackToV2(t *testing.T) {
	// Verify that configuring processor_version: "v1" logs a warning
	// and creates a v2 input without error.
	logp.TestingSetup(logp.WithSelectors("azureeventhub"))
	log := logp.NewLogger("azureeventhub")

	manager := &eventHubInputManager{log: log}

	config := conf.MustNewConfigFrom(map[string]interface{}{
		"eventhub":                         "test-hub",
		"connection_string":                "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test",
		"storage_account":                  "teststorage",
		"storage_account_connection_string": "DefaultEndpointsProtocol=https;AccountName=teststorage;AccountKey=secret;EndpointSuffix=core.windows.net",
		"processor_version":                "v1",
	})

	input, err := manager.Create(config)
	require.NoError(t, err)
	require.NotNil(t, input)

	// Verify the input is a v2 input
	_, ok := input.(*eventHubInputV2)
	assert.True(t, ok, "expected eventHubInputV2 when processor_version is v1")
}
```

This requires adding imports: `conf "github.com/elastic/elastic-agent-libs/config"`, `"github.com/elastic/elastic-agent-libs/logp"`, `"github.com/stretchr/testify/assert"`, `"github.com/stretchr/testify/require"`, and `"testing"`.

- [ ] **Step 8: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/input_test.go
git commit -m "azureeventhub: remove v1 tests, add v1→v2 fallback test

Remove TestGetAzureEnvironment, TestProcessEvents, commented-out
TestNewInputDone, and defaultTestConfig. Clean up deprecated SDK imports.
Add TestCreateWithProcessorV1FallsBackToV2 to verify the deprecation
warning path. Keep fakeClient for use by metrics_test.go."
```

---

### Task 6: Rewrite metrics_test.go to use messageDecoder directly

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/metrics_test.go`

The current test creates an `eventHubInputV1{}` and calls `processEvents()` with legacy `eventhub.Event` types. The test is really validating that metrics are correctly updated during message decode and publish. We can rewrite it to use `messageDecoder.Decode()` directly and publish via `fakeClient`, which tests the same metric behavior without v1 types.

- [ ] **Step 1: Replace imports**

Replace the imports with:
```go
import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)
```

- [ ] **Step 2: Rewrite TestInputMetricsEventsReceived**

Rewrite the test to use `messageDecoder` directly. For each test case:
1. Create a `messageDecoder` with the test config and metrics
2. Call `decoder.Decode(tc.event)` to get records
3. Publish events via `fakeClient` (to match the original flow)
4. Verify metrics

```go
func TestInputMetricsEventsReceived(t *testing.T) {
	log := logp.NewLogger("azureeventhub test for input")

	cases := []struct {
		name string
		// Use case definition
		event              []byte
		expectedRecords    []string
		sanitizationOption []string
		// Expected results
		receivedMessages    uint64
		invalidJSONMessages uint64
		sanitizedMessages   uint64
		processedMessages   uint64
		receivedEvents      uint64
		sentEvents          uint64
		decodeErrors        uint64
	}{
		{
			name:                "single valid record",
			event:               []byte("{\"records\": [{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"),
			expectedRecords:     []string{"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"},
			receivedMessages:    1,
			invalidJSONMessages: 0,
			sanitizedMessages:   0,
			processedMessages:   1,
			receivedEvents:      1,
			sentEvents:          1,
			decodeErrors:        0,
		},
		{
			name:  "two valid records",
			event: []byte("{\"records\": [{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}, {\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"),
			expectedRecords: []string{
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
			},
			receivedMessages:    1,
			invalidJSONMessages: 0,
			sanitizedMessages:   0,
			processedMessages:   1,
			receivedEvents:      2,
			sentEvents:          2,
			decodeErrors:        0,
		},
		{
			name:  "single quotes sanitized",
			event: []byte("{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}"),
			expectedRecords: []string{
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
			},
			sanitizationOption:  []string{"SINGLE_QUOTES"},
			receivedMessages:    1,
			invalidJSONMessages: 1,
			sanitizedMessages:   1,
			processedMessages:   1,
			receivedEvents:      1,
			sentEvents:          1,
			decodeErrors:        0,
		},
		{
			name:  "invalid JSON without sanitization returns raw message",
			event: []byte("{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}"),
			expectedRecords: []string{
				"{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}",
			},
			sanitizationOption:  []string{},
			receivedMessages:    1,
			invalidJSONMessages: 1,
			sanitizedMessages:   0,
			processedMessages:   1,
			decodeErrors:        1,
			receivedEvents:      0,
			sentEvents:          1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inputConfig := azureInputConfig{
				SAName:                "",
				SAContainer:           ephContainerName,
				ConnectionString:      "",
				ConsumerGroup:         "",
				LegacySanitizeOptions: tc.sanitizationOption,
			}

			metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

			client := fakeClient{}

			sanitizers, err := newSanitizers(inputConfig.Sanitizers, inputConfig.LegacySanitizeOptions)
			require.NoError(t, err)

			decoder := messageDecoder{
				config:     inputConfig,
				metrics:    metrics,
				log:        log,
				sanitizers: sanitizers,
			}

			// Simulate the processing pipeline: decode + publish
			metrics.receivedMessages.Inc()
			metrics.receivedBytes.Add(uint64(len(tc.event)))

			records := decoder.Decode(tc.event)
			for _, record := range records {
				event := beat.Event{
					Fields: mapstr.M{
						"message": record,
					},
				}
				client.Publish(event)
			}
			metrics.processedMessages.Inc()
			metrics.sentEvents.Add(uint64(len(records)))

			// Verify published events
			if ok := assert.Equal(t, len(tc.expectedRecords), len(client.publishedEvents)); ok {
				for i, e := range client.publishedEvents {
					msg, err := e.Fields.GetValue("message")
					if err != nil {
						t.Fatal(err)
					}
					assert.Equal(t, msg, tc.expectedRecords[i])
				}
			}

			// Messages
			assert.Equal(t, tc.receivedMessages, metrics.receivedMessages.Get())
			assert.Equal(t, uint64(len(tc.event)), metrics.receivedBytes.Get())
			assert.Equal(t, tc.invalidJSONMessages, metrics.invalidJSONMessages.Get())
			assert.Equal(t, tc.sanitizedMessages, metrics.sanitizedMessages.Get())
			assert.Equal(t, tc.processedMessages, metrics.processedMessages.Get())

			// General
			assert.Equal(t, tc.decodeErrors, metrics.decodeErrors.Get())

			// Events
			assert.Equal(t, tc.receivedEvents, metrics.receivedEvents.Get())
			assert.Equal(t, tc.sentEvents, metrics.sentEvents.Get())
		})
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/metrics_test.go
git commit -m "azureeventhub: rewrite metrics_test to use messageDecoder directly

Replace eventHubInputV1 and legacy eventhub.Event usage with
messageDecoder.Decode() to test the same metrics behavior
without depending on the deleted v1 types."
```

---

### Task 7: Clean up v1 comment in client_secret.go

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/client_secret.go:79-81`

- [ ] **Step 1: Update the comment**

Change line 81 from:
```go
// the deprecated go-autorest package. For processor v1, use getAzureEnvironment() instead.
```
to:
```go
// the deprecated go-autorest package.
```

- [ ] **Step 2: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/client_secret.go
git commit -m "azureeventhub: remove v1 reference from client_secret.go comment"
```

---

### Task 8: Build and test

- [ ] **Step 1: Run the package tests**

```bash
cd x-pack/filebeat/input/azureeventhub
go test ./...
```

Expected: All tests pass with no compilation errors.

- [ ] **Step 2: Build filebeat**

```bash
cd x-pack/filebeat
go build ./...
```

Expected: Build succeeds.

- [ ] **Step 3: Fix any compilation or test failures**

If tests fail, fix the issues. Common issues might include:
- Unused imports that need removing
- Missing type references
- Test assertions that need updating

- [ ] **Step 4: Commit any fixes**

```bash
git add x-pack/filebeat/input/azureeventhub/
git commit -m "azureeventhub: fix compilation/test issues after v1 removal"
```

---

### Task 9: Update documentation

**Files:**
- Modify: `x-pack/filebeat/input/azureeventhub/README.md`
- Modify: `docs/reference/filebeat/filebeat-input-azure-eventhub.md`

- [ ] **Step 1: Update README.md**

In the README, the "Start with v1" section (around line 265) and all v1 migration testing instructions are historical context for how to test the v1→v2 migration. Since v1 is no longer available, update the migration testing section to note that v1 is no longer available. Remove or mark the v1-specific instructions as historical. Keep the v2 checkpoint migration testing instructions since `migrate_checkpoint` is still active.

- [ ] **Step 2: Update docs/reference/filebeat/filebeat-input-azure-eventhub.md**

**Remove the tracing paragraph** (line 16). After removing `tracer.go` and the `tab.Register` call, the `BEATS_AZURE_EVENTHUB_INPUT_TRACING_ENABLED` environment variable no longer has any effect. Remove or update this paragraph:
```markdown
Enable internal logs tracing for this input by setting the environment variable `BEATS_AZURE_EVENTHUB_INPUT_TRACING_ENABLED: true`. ...
```

The v2 input has its own tracing via the status reporter. If v2 has equivalent tracing functionality, update the paragraph to describe it. Otherwise, remove it entirely.

**Remove the v1 example section** (lines 20-36):
```markdown
### Connection string authentication (processor v1)

**Note:** Processor v1 only supports connection string authentication.

Example configuration using connection string authentication with processor v1:

...
```

**Remove "(processor v2)" from remaining section headings:**
- "Connection string authentication (processor v2)" → "Connection string authentication"
- "Client secret authentication (processor v2)" → "Client secret authentication"
- "Managed identity authentication (processor v2)" → "Managed identity authentication"

Also remove "(processor v2)" and "with processor v2" from description text in those sections.

**Update the intro paragraph** (line 12). Replace the reference to the deprecated EPH SDK:
```markdown
Use the `azure-eventhub` input to read messages from an Azure EventHub. The azure-eventhub input implementation is based on the event processor host. EPH is intended to be run across multiple processes and machines while load balancing message consumers more on this here [https://github.com/Azure/azure-event-hubs-go#event-processor-host](https://github.com/Azure/azure-event-hubs-go#event-processor-host), [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host).
```

With:
```markdown
Use the `azure-eventhub` input to read messages from an Azure Event Hub. The input uses the [Azure Event Hubs SDK for Go](https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/messaging/azeventhubs) to consume events with load-balanced partition processing across multiple instances.
```

**Update `storage_account_key` description** (line 264). Change from:
```markdown
The storage account key, this key will be used to authorize access to data in your storage account, option is required.
```
to:
```markdown
The storage account key. When using `connection_string` authentication, you can provide either `storage_account_connection_string` (recommended) or `storage_account_key` together with `storage_account` to auto-construct the connection string. Not required when using `client_secret` or `managed_identity` authentication.
```

- [ ] **Step 3: Commit**

```bash
git add x-pack/filebeat/input/azureeventhub/README.md docs/reference/filebeat/filebeat-input-azure-eventhub.md
git commit -m "docs: update azure-eventhub documentation to reflect v1 removal

- Remove v1 example section from reference docs
- Remove '(processor v2)' suffixes from section headings
- Update intro paragraph to reference modern SDK
- Update storage_account_key description
- Update README migration testing instructions"
```

---

### Task 10: Clean up Go module dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 2: Check if deprecated packages were removed**

```bash
grep -E "azure-event-hubs-go|azure-storage-blob-go|devigned/tab" go.mod
```

If any remain, they are transitive dependencies from other Beats packages. Document this for follow-up.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "azureeventhub: run go mod tidy after v1 removal

Clean up Go module dependencies after removing processor v1
and its deprecated Azure SDK imports."
```

---

### Task 11: Final verification

- [ ] **Step 1: Run the full package test suite**

```bash
cd x-pack/filebeat/input/azureeventhub
go test -v ./...
```

Expected: All tests pass.

- [ ] **Step 2: Build filebeat**

```bash
cd x-pack/filebeat
go build ./...
```

Expected: Build succeeds.

- [ ] **Step 3: Run go vet**

```bash
cd x-pack/filebeat/input/azureeventhub
go vet ./...
```

Expected: No issues.
