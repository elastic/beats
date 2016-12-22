# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added
- Added `CpuList` implementation for Windows that returns CPU timing information
  on a per CPU basis. #55
- Added `Uptime` implementation for Windows. #55
- Added `Swap` implementation for Windows based on page file metrics. #55
- Added support to `github.com/gosigar/sys/windows` for querying and enabling
  privileges in a process token.
- Added utility code for interfacing with linux NETLINK_INET_DIAG. #60

### Changed
- Changed several `OpenProcess` calls on Windows to request the lowest possible
  access privileges. #50
- Removed cgo usage from Windows code.
- Added OS version checks to `ProcArgs.Get` on Windows because the
  `Win32_Process` WMI query is not available prior to Windows vista. On XP and
  Windows 2003, this method returns `ErrNotImplemented`. #55

### Deprecated

### Removed

### Fixed
- Fixed value of `Mem.ActualFree` and `Mem.ActualUsed` on Windows. #49
- Fixed `ProcTime.StartTime` on Windows to report value in milliseconds since
  Unix epoch. #51
- Fixed `ProcStatus.PPID` value is wrong on Windows. #55
- Fixed `ProcStatus.Username` error on Windows XP #56
