---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/recording-trace.html
applies_to:
  stack: ga
---

# Record a trace [recording-trace]

If you are having an issue, it’s often useful to record a full network trace and send it to us. It will help us reproduce the issue, and we can also add it to our automatic regression tests so that the problem never reoccurs. A trace of 10-20 seconds is usually enough. To record the trace, you can use the following Packetbeat command:

```shell
packetbeat -e --dump trace.pcap
```

This command executes Packetbeat in normal mode (all processing happens as usual), but at the same time, it records all packets in libpcap format in the `trace.pcap` file. If there’s a particular error message you want us to investigate, please keep the trace running until the error shows up (it will printed on standard error).

::::{warning}
PCAP files can be large. Please monitor the disk usage while doing the dump to make sure you don’t run out of disk space. Whenever possible, we recommend recording the trace on a non-production machine.
::::


