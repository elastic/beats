# go-sysinfo

[![Build Status](http://img.shields.io/travis/elastic/go-sysinfo.svg?style=flat-square)][travis]
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[travis]: http://travis-ci.org/elastic/go-sysinfo
[godocs]: http://godoc.org/github.com/elastic/go-sysinfo

go-sysinfo is a library for collecting system information. This includes
information about the host machine and processes running on the host.

The available features vary based on what has been implemented by the "provider"
for the operating system. At runtime you check to see if additional interfaces
are implemented by the returned `Host` or `Process`. For example:

```go
process, err := sysinfo.Self()
if err != nil {
	return err
}

if handleCounter, ok := process.(types.OpenHandleCounter); ok {
	count, err := handleCounter.OpenHandleCount()
	if err != nil {
		return err
	}
	log.Printf("%d open handles", count)
}
```

These tables show what methods are implemented as well as the extra interfaces
that are implemented.

| `Host` Features  | Darwin | Linux | Windows |
|------------------|--------|-------|---------|
| `Info()`         | x      | x     | x       |
| `Memory()`       | x      | x     | x       |
| `CPUTimer`       | x      | x     | x       |
| `VMStat`         |        | x     |         |

| `Process` Features     | Darwin | Linux | Windows |
|------------------------|--------|-------|---------|
| `Info()`               | x      | x     | x       |
| `Memory()`             | x      | x     | x       |
| `User()`               | x      | x     | x       |
| `Parent()`             | x      | x     | x       |
| `CPUTimer`             | x      | x     | x       |
| `Environment`          | x      | x     |         |
| `OpenHandleEnumerator` |        | x     |         |
| `OpenHandleCounter`    |        | x     |         |
| `Seccomp`              |        | x     |         |
| `Capabilities`         |        | x     |         |


