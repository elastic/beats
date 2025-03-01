# Integration Framework for Beats

This package contains a simple framework for integration testing of Beats. The main goal of the framework is to make it easy to test a Beat binary as close to our customer's usage as possible. No custom binaries, no inspecting internal state files, just pure output and asserting external behavior like if the Beat was a black box.

## Current functionality

### Basic Assertions

* Assert an output line that contains a defined string
* Assert a list of output lines in a defined order that contain a given list of strings
* Assert an output line that matches a regular expression
* Assert a list of output lines in a defined order that match a given list of regular expressions
* Assert that the process started
* Assert that the process exited by itself with a certain exit code

When building a Beat-specific wrapper around this framework, new assertions can be created based on the basic assertions listed above.

### Reporting

* Print out all defined expectations of the test
* Print last `N` lines of the output

### Config

* Add additional arguments to the command to run the binary
* Pass a config file (e.g. `filebeat.yml`)

## Quick start

### Things to know before you start

This framework:

* does not use log files for inspecting/matching the expected logs. Instead it connects directly to stdout/stderr and matches all the output expectations in memory line by line as they arrive. Which makes it extremely efficient at expecting thousands of log lines (e.g. confirming each line of a file gets ingested).
* kills the process immediately once the defined expectations are met, no more polling with intervals.
* runs the binary that we ship to our customers instead of a [custom binary](https://github.com/elastic/beats/blob/12c36bdfa6fe088f3963bdf5e15780878c228eaf/dev-tools/mage/gotest.go#L399-L430)
* has a call-chain interface which is very compact
* supports testing cases when a Beat crashes with errors
* has very detailed output for debugging a test failure
* is generic and in theory can be used with any Beat
* can be extended and specialized for each Beat, see the [example with Filebeat](https://github.com/elastic/beats/tree/main/filebeat/testing/integration).

### Samples

Sample test that validates that Filebeat started, read all the expected files to EOF and ingested all the lines from them:

```go
func TestFilebeat(t *testing.T) {
	messagePrefix := "sample text message"
	fileCount := 5
	lineCount := 128
	configTemplate := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
# we want to check that all messages are ingested
# without using an external service, this is an easy way
output.console:
  enabled: true
`
	// we can generate any amount of expectations
	// they are light-weight
	expectIngestedFiles := func(test Test, files []string) {
		// ensuring we ingest every line from every file
		for _, filename := range files {
			for i := 1; i <= lineCount; i++ {
				line := fmt.Sprintf("%s %s:%d", messagePrefix, filepath.Base(filename), i)
				test.ExpectOutput(line)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	generator := NewJSONGenerator(messagePrefix)
	path, files := GenerateLogFiles(t, fileCount, lineCount, generator)
	config := fmt.Sprintf(configTemplate, path)
	test := NewTest(t, TestOptions{
		Config: config,
	})

	expectIngestedFiles(test, files)

	test.
		ExpectEOF(files...).
		ExpectStart().
		Start(ctx).
		Wait()
}
```

Another sample test, this time we expect Beat to crash:

```go
func TestFilebeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// path items are required, this config is invalid
	config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
output.console:
  enabled: true
`
	test := NewBeatTest(t, BeatTestOptions{
		Beatname: "filebeat",
		Config:   config,
	})

	test.
		ExpectStart().
		ExpectOutput("Exiting: Failed to start crawler: starting input failed: error while initializing input: no path is configured").
		ExpectStop(1).
		Start(ctx).
		Wait()
}
```
