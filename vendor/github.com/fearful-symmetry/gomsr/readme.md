# gomsr

[![GoDoc](https://godoc.org/github.com/fearful-symmetry/gomsr?status.svg)](https://godoc.org/github.com/fearful-symmetry/gomsr)
[![Go Report Card](https://goreportcard.com/badge/github.com/fearful-symmetry/gomsr)](https://goreportcard.com/report/github.com/fearful-symmetry/gomsr)
[![CircleCI](https://circleci.com/gh/fearful-symmetry/gomsr.svg?style=svg)](https://circleci.com/gh/fearful-symmetry/gomsr)

`gomsr` is a library for reading from and writing to [MSRs](https://en.wikipedia.org/wiki/Model-specific_register). It'll work on any linux machine that supports the `msr` kernel module. It also supports setting a custom character device location, allowing `gomsr` to work with utilities such as [msr-safe](https://github.com/llnl/msr-safe).

A quick and dirty check to see if this will work on your system:

```bash
$ ls /dev/cpu/0/msr
/dev/cpu/0/msr

# on some systems, you might need to do this first
$ sudo modprobe msr
```

This library is a new WIP, and breaking changes should be expected.


## Usage

`gomsr` has no dependencies, and is super easy to use:

```go

//0x198 is IA32_PERF_STATUS on most Intel CPUs

//ReadMSR() takes a CPU and an MSR address
data, err := ReadMSR(0, 0x198)
if err != nil {
	log.Fatalf("Error: %s", err)
}

//You can also write to MSRs
err := WriteMSR(0, 0x401, 0)
if err != nil {
	log.Fatalf("Error: %s", err)
}

//You can also create an msr handler. This is suited for repeated reads/writes

//the MSR() init function just takes a CPU
msr, err := MSR(0)
if err != nil {
	log.Fatalf("Error: %s", err)
}

data, err := msr.Read(0x610)
if err != nil {
	log.Fatalf("Error: %s", err)
}
fmt.Printf("Got 0x%x\n", data)

msr.Close()
```