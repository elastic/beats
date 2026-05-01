# Goal
We need to write a thorough set of tests for Filestream's TakeOver
mode, the currently fallback mechanism implemented is:
 - Filestream migrates the states from the Log input (see
   fileProspector.TakeOver
   @filebeat/input/filestream/prospector.go:292)
 - Filestream copies the Log input state to its format, leaving the Log
   input state untouched
 - Once Filestream actually starts running, the states are completely
   isolated
 - If the Filestream input is disabled and the Log input is
   re-enabled, then the Log input will continue from where it had left
   off and both inputs will have independent states for the files.

Our goal is to ensure this state isolation and that the Filestream
input does not alter in any way the Log input state.

## 1. The data generator
We need to extend the data generation WriteLogFile
@libbeat/tests/integration/datagenerator.go:39 so it can accept
a "start at" parameter. The idea is: each log line contains an
ever-increasing counter and we can run the function multiple times and
have a sequence that never restarts. The best option is to create a
new exported function and re-use most of the existing code in both
functions (the existing and the new one)


## 2. The first test
We will use Filebeat's input reload feature to exercise a more dynamic
environment/workflow.

Use the file output, every time we stop all inputs, make a copy of the
output file, so we can analyse what was the output at each step.

We will write a integration test with the following flow:
 - Start data generation on two different files
 - Start Filebeat with no inputs in the inputs.d folder
 - Run the log input
 - Stop Filebeat
 - Assert the correct number of events was ingested
 - Replace the Log input by Filestream with take over enabled
 - Run for a little while 
 - Stop Filestream
 - Assert the correct number of events was ingested (no data
   duplication)
 - Start the Log input (simulate the fallback)
 - Run fro a little while
 - Stop the Log input
 - Assert it continued from where it had left off
 - Start Filestream again
 - Let it run for a little while
 - Stop Filestream
 - Assert Filestream continued ingesting data from where it had left
   off and no duplication happened.

To differentiate events from each input, we can filter them by
`input.type`:
 - Filestream: `input.type: filestream`
 - Log: `input.type: log`

For the data generation we will use WriteLogFileFrom
(@libbeat/tests/integration/datagenerator.go:48), the prefix will be
set in a way that:
 - A human reviewing the output file can easily tell which file/step
   the log line belongs
 - Each log line is always 100 characters long
