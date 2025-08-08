# Beats Integration tests framework

This integration test framework aims to replace our Python one, it
started small by implementing in Go the basic functionality and
helpers provided by the Python one and since them it has been growing
in features and helper functions.

## Beat Process Creation and Management

### Beat Creation
+ ~NewBeat~: Creates a new Beat process from the system tests binary with specific settings.
+ ~NewStandardBeat~: Creates a Beat process from a standard binary without the system test flag.
+ ~NewAgentBeat~: Creates a new agentbeat process that runs the beatName as a subcommand.

#### Process Control
+ ~Start~: Starts the Beat process with optional additional arguments.
+ ~Stop~: Stops the Beat process (framework adds cleanup to stop automatically at test end).
+ ~RestartOnBeatOnExit~: When set to true, automatically restarts the Beat when it exits.

#### Configuration Management
+ ~WriteConfigFile~: Writes provided configuration string to a file for the Beat to use.
+ ~ConfigFilePath~: Returns the path to the Beat's configuration file.
+ ~RemoveAllCLIArgs~: Removes all CLI arguments, keeping only the required ~--systemTest~ flag.

## Reading and Writing log files

### Methods search in logs keeping track of the lines already read
These methods remember what they've already read and only check new content on subsequent calls:

+ ~LogContains~: Checks if logs contain a string (keeps track of offset).
+ ~LogMatch~: Tests if logs match a regular expression (keeps track of offset).
+ ~WaitLogsContains~: Waits for logs to contain a string (uses ~LogContains~ with offset tracking).

### Methods search in logs without tracking of the lines already read
These methods always read from the beginning of logs:

+ ~GetLogLine~: Searches for a string in logs and returns the matching line.
+ ~GetLastLogLine~: Searches for a string from the end of logs and returns the matching line.
+ ~WaitLogsContainsFromBeginning~: Resets offset, then waits for logs to contain a string.
+ ~WaitForLogsAnyOrder~ - Waits for all specified strings to appear in logs in any order within the given timeout.

### Generating log files for ingestion

+ ~WriteLogFile~ - Writes a log file with the specified number of lines, each containing timestamp and a counter.
+ ~WriteNLogFiles~ - Generates multiple log files (nFiles) each containing the specified number of random lines (nLines).
+ ~WriteAppendingLogFile~ - Generates a log file by appending the current time to it every second.

## File Operations

+ ~FileContains~: Searches for a string in a file and returns the first matching line.
+ ~WaitFileContains~: Waits for a file to contain a specific string.
+ ~WaitStdErrContains~: Waits for stderr to contain a specific string.
+ ~WaitStdOutContains~: Waits for stdout to contain a specific string.
+ ~WaitPublishedEvents~ - Waits until the desired number of events have been published to the file output.
+ ~CountFileLines~: Counts the number of lines in files matching a glob pattern.
+ ~RemoveLogFiles~: Removes log files and resets the tracked offsets.

## Reading data from the Output

+ ~GetEventsMsgFromES~ - Retrieves event messages from Elasticsearch with specified query parameters.
+ ~GetEventsFromFileOutput~ - Reads events from output files in the specified directory with optional limit on number of events.

## Utilities

+ ~TempDir~: Returns the temporary directory used by the Beat (preserved on test failure).
+ ~LoadMeta~: Loads metadata from the Beat's meta.json file.
+ ~Stdin~: Returns the standard input for interacting with the Beat process.
+ ~WaitLineCountInFile~: Waits until a file has a specific number of lines.

### Elasticsearch Integration

#### Connection
+ ~EnsureESIsRunning~: Checks that Elasticsearch is running and accessible.
+ ~GetESClient~: Gets an Elasticsearch client configured with test credentials.
+ ~GetESURL~: Returns the Elasticsearch URL with standard user credentials.
+ ~GetESAdminURL~: Returns the Elasticsearch URL with admin credentials.

#### Mock Elasticsearch
+ ~StartMockES~: Starts a mock Elasticsearch server with configurable behavior.

### HTTP Requests
+ ~HttpDo~: Performs an HTTP request to a specified URL.
+ ~FormatDatastreamURL~: Formats a URL for a datastream endpoint.
+ ~FormatIndexTemplateURL~: Formats a URL for an index template endpoint.
+ ~FormatPolicyURL~: Formats a URL for a policy endpoint.
+ ~FormatRefreshURL~: Formats a URL for refreshing indices.
+ ~FormatDataStreamSearchURL~: Formats a URL for searching a datastream.

### Kibana Integration
+ ~GetKibana~: Returns the Kibana URL and user credentials.
