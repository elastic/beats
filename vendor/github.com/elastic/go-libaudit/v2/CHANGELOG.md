# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Removed

### Deprecated

## [2.2.0]

### Added

- Add user and group mapping for ECS 1.8 compatibility [#86](https://github.com/elastic/go-libaudit/pull/86)

### Changed

- Change ECS category of USER_START and USER_END messages to `session`. [#86](https://github.com/elastic/go-libaudit/pull/86)

## [2.1.0]

### Added

- ECS 1.7 `configuration` categorization. [#80](https://github.com/elastic/go-libaudit/pull/80)

### Changed

- Use ingress/egress instead of inbound/outbound for ECS 1.7. [#80](https://github.com/elastic/go-libaudit/pull/80)

## [2.0.2]

### Changed

- Use ECS recommended values for network direction. [#75](https://github.com/elastic/go-libaudit/issues/75)[#76](https://github.com/elastic/go-libaudit/pull/76)
  
### Removed

- Remove github.com/Sirupsen/logrus dependency from examples. [#73](https://github.com/elastic/go-libaudit/issues/73)


## [2.0.1]

### Changed

- Fixed syscall lookup for ppc64 and ppc64le. [#71](https://github.com/elastic/go-libaudit/pull/71)

## [2.0.0]

### Added

- Added `SetImmutable` to the audit client for marking the audit settings as immutable within the kernel. [#55](https://github.com/elastic/go-libaudit/issue/55) [#68](https://github.com/elastic/go-libaudit/pull/68)
- Added Vagrantfile for development ease. [#61](https://github.com/elastic/go-libaudit/pull/61)
- Added enrichment of arch, syscall, and sig to type=SECCOMP messages. [#64](https://github.com/elastic/go-libaudit/pull/64)
- Added support for big endian. [#48](https://github.com/elastic/go-libaudit/pull/48)

### Changed

- Added semantic versioning support via go modules. [#61](https://github.com/elastic/go-libaudit/pull/61)
- Added ECS categorization support for events by record type and syscall. [#62](https://github.com/elastic/go-libaudit/pull/62)
- Fixed a typo in the action value associated with ROLE_REMOVE messages. [#65](https://github.com/elastic/go-libaudit/pull/65)
- Fixed a typo in the action value associated with ANOM_LINK messages. [#66](https://github.com/elastic/go-libaudit/pull/66)
- Fixed spelling of anomaly in aucoalesce package. [#67](https://github.com/elastic/go-libaudit/pull/67)

## [0.4.0]

### Added

- Added method to convert kernel rules to text format in order to display them.

### Changed

- aucoalesce - Made the user/group ID cache thread-safe. [#42](https://github.com/elastic/go-libaudit/pull/42) [#45](https://github.com/elastic/go-libaudit/pull/45)

## [0.3.0]

### Added

- Added support for setting the kernel's backlog wait time via the new
  SetBacklogWaitTime function. [#34](https://github.com/elastic/go-libaudit/pull/34)
- New method `GetStatusAsync` to perform asynchronous status checks. [#37](https://github.com/elastic/go-libaudit/pull/37)

### Changed

- AuditClient `Close()` is now safe to call more than once. [#35](https://github.com/elastic/go-libaudit/pull/35)

## [0.2.1]

### Added

- Added better error messages for when `NewAuditClient` fails due to the
  Linux kernel not supporting auditing (CONFIG_AUDIT=n). [#32](https://github.com/elastic/go-libaudit/pull/32)

## [0.2.0]

### Changed

- auparse - Fixed parsing of apparmor AVC messages. [#25](https://github.com/elastic/go-libaudit/pull/25)
- auparse - Update syscall and audit message type tables for Linux 4.16.
- aucoalesce - Cache UID/GID values for one minute. [#24](https://github.com/elastic/go-libaudit/pull/24)

## [0.1.1]

- rules - Detect s390 or s390x as the runtime architecture (GOOS) and
  automatically use the appropriate syscall name to number table without
  requiring the rule to explicitly specify an arch (`-F arch=s390x`). [#23](https://github.com/elastic/go-libaudit/pull/23)

## [0.1.0]

### Changed

- auparse - Fixed an issue where the name value was not being hex decoded from
  PATH records. [#20](https://github.com/elastic/go-libaudit/pull/20)

## [0.0.7]

### Added

- Added WaitForPendingACKs to receive pending ACK messages from the kernel. [#14](https://github.com/elastic/go-libaudit/pull/14)
- The AuditClient will unregister with the kernel if `SetPID` has been called. [#19](https://github.com/elastic/go-libaudit/pull/19)

### Changed

- auparse - Fixed an issue where the proctitle value was being truncated. [#15](https://github.com/elastic/go-libaudit/pull/15)
- auparse - Fixed an issue where values were incorrectly interpretted as hex
  data. [#13](https://github.com/elastic/go-libaudit/pull/13)
- auparse - Fixed parsing of the `key` value when multiple keys are present. [#16](https://github.com/elastic/go-libaudit/pull/16)
- auparse - The `cmdline` key is no longer created for EXECVE records. [#17](https://github.com/elastic/go-libaudit/pull/17)
- aucoalesce - Changed the event format to have objects for user, process, file,
  and network data. [#17](https://github.com/elastic/go-libaudit/pull/17)
- Fixed an issue when an audit notification is received while waiting for the
  response to a control command.

## [0.0.6]

### Added

- Add support for listening for audit messages using a multicast group. [#9](https://github.com/elastic/go-libaudit/pull/9)

## [0.0.5]

### Changed

- auparse - Apply hex decoding to CWD field. [#10](https://github.com/elastic/go-libaudit/pull/10)

## [0.0.4]

### Added

- Add a package for building audit rules that can be added to the kernel.
- Add GetRules, DeleteRules, DeleteRule, and AddRule methods to AuditClient.
- auparse - Add conversion of POSIX exit code values to their name.
- Add SetFailure to AuditClient. [#8](https://github.com/elastic/go-libaudit/pull/8)

## [0.0.3]

### Added

- auparse - Convert auid and session values of `4294967295` or `-1` to "unset". [#5](https://github.com/elastic/go-libaudit/pull/5)
- auparse - Added `MarshallText` method to AuditMessageType to enable the value
  to be marshaled as a string in JSON. faabfa94ec9479bdc1ad6c0334ff178b8193fce5
- aucoalesce - Enhanced aucoalesce to normalize events. 666ff1c30fe624e9fcd9a108b20fceb82331f5fa

### Changed

- Rename RawAuditMessage fields `MessageType` and `RawData` to `Type` and
  `Data` respectively. 8622833714fccd7810669b1265df1c1f918ec0c4
- Make Reassembler concurrency-safe. c57b59c20a684e2a6298a1a5929a79192d76d61b
- auparse - Renamed `address_family` to `family` in parsed sockaddr messages.
  73f97b2f366e6e00acf2ddff4f6575432da3283e

## [0.0.2]

### Added

- Added `libaudit.Reassembler` for reassembling out of order or interleaved
  messages and providing notification for lost events based on gaps in sequence
  numbers. a60bdd3b1b642cc80a3872d999114ae675456768
- auparse - Combine EXECVE arguments into a single field called `cmdline`.
  468a9eb0898e0efd3c2fd7abf067519cb63fa6c3
- auparse - Split SELinux subjects into `subj_user`, `subj_role`,
  `subj_domain`, `subj_level`, and `subj_category`.
  f3ed884a7c03ea75c9ec247251905aa1ec548959
- auparse - Replace auid values `4294967295` and `-1` with `unset` to convey
  the meaning of these values. [#5](https://github.com/elastic/go-libaudit/pull/5)
- aucoalesce - Added a new package to coalescing related messages into a single
  event. [#1](https://github.com/elastic/go-libaudit/pull/1)

### Changed

- auparse - Changed the behavior of `ParseLogLine()` and `Parse()` to only parse
  the message header. To parse the message body, call `Data()` on the returned
  `AuditMessage`.

## [0.0.1]

### Added

- Added AuditClient for communicating with the Linux Audit Framework in the
  Linux kernel.
- Added auparse package for parsing audit logs.

[Unreleased]: https://github.com/elastic/go-libaudit/compare/v2.1.0...HEAD
[2.1.0]: https://github.com/elastic/go-libaudit/compare/v2.1.0
[2.0.2]: https://github.com/elastic/go-libaudit/releases/tag/v2.0.2
[2.0.1]: https://github.com/elastic/go-libaudit/releases/tag/v2.0.1
[2.0.0]: https://github.com/elastic/go-libaudit/releases/tag/v2.0.0
[0.4.0]: https://github.com/elastic/go-libaudit/releases/tag/v0.4.0
[0.3.0]: https://github.com/elastic/go-libaudit/releases/tag/v0.3.0
[0.2.1]: https://github.com/elastic/go-libaudit/releases/tag/v0.2.1
[0.2.0]: https://github.com/elastic/go-libaudit/releases/tag/v0.2.0
[0.1.1]: https://github.com/elastic/go-libaudit/releases/tag/v0.1.1
[0.1.0]: https://github.com/elastic/go-libaudit/releases/tag/v0.1.0
[0.0.7]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.7
[0.0.6]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.6
[0.0.5]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.5
[0.0.4]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.4
[0.0.3]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.3
[0.0.2]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.2
[0.0.1]: https://github.com/elastic/go-libaudit/releases/tag/v0.0.1
