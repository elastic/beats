# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [0.6.0]

### Added
- Added method stubs to enable compilation for operating systems that are not
  supported by gosigar. All methods return `ErrNotImplemented` on these unsupported
  operating systems. #83
- FreeBSD returns `ErrNotImplemented` for `ProcTime.Get`. #83

### Changed
- OpenBSD returns `ErrNotImplemented` for `ProcTime.Get` instead of `nil`. #83
- Fixed incorrect `Mem.Used` calculation under linux. #82
- Fixed `ProcState` on Linux and FreeBSD when process names contain parentheses. #81

### Removed
- Remove NetBSD build from sigar_unix.go as it is not supported by gosigar. #83

## [0.5.0]

### Changed
- Fixed Trim environment variables when comparing values in the test suite. #79
- Make `kern_procargs` more robust under darwin when we cannot retrieve
  all the information about a process. #78

## [0.4.0]

### Changed
- Fixed Windows issue that caused a hang during `init()` if WMI wasn't ready. #74

## [0.3.0]

### Added
- Read `MemAvailable` value for kernel 3.14+ #71

## [0.2.0]

### Added
- Added `ErrCgroupsMissing` to indicate that /proc/cgroups is missing which is
  an indicator that cgroups were disabled at compile time. #64

### Changed
- Changed `cgroup.SupportedSubsystems()` to honor the "enabled" column in the
  /proc/cgroups file. #64

## [0.1.0]

### Added
- Added `CpuList` implementation for Windows that returns CPU timing information
  on a per CPU basis. #55
- Added `Uptime` implementation for Windows. #55
- Added `Swap` implementation for Windows based on page file metrics. #55
- Added support to `github.com/gosigar/sys/windows` for querying and enabling
  privileges in a process token.
- Added utility code for interfacing with linux NETLINK_INET_DIAG. #60
- Added `ProcEnv` for getting a process's environment variables. #61

### Changed
- Changed several `OpenProcess` calls on Windows to request the lowest possible
  access privileges. #50
- Removed cgo usage from Windows code.
- Added OS version checks to `ProcArgs.Get` on Windows because the
  `Win32_Process` WMI query is not available prior to Windows vista. On XP and
  Windows 2003, this method returns `ErrNotImplemented`. #55

### Fixed
- Fixed value of `Mem.ActualFree` and `Mem.ActualUsed` on Windows. #49
- Fixed `ProcTime.StartTime` on Windows to report value in milliseconds since
  Unix epoch. #51
- Fixed `ProcStatus.PPID` value is wrong on Windows. #55
- Fixed `ProcStatus.Username` error on Windows XP #56
