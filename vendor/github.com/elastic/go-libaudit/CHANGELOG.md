# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [0.0.6]

### Added

- Add support for listening for audit messages using a multicast group. #9

## [0.0.5]

### Changed
- auparse - Apply hex decoding to CWD field. #10

## [0.0.4]

### Added
- Add a package for building audit rules that can be added to the kernel.
- Add GetRules, DeleteRules, DeleteRule, and AddRule methods to AuditClient.
- auparse - Add conversion of POSIX exit code values to their name.
- Add SetFailure to AuditClient. #8

## [0.0.3]

### Added
- auparse - Convert auid and session values of `4294967295` or `-1` to "unset". #5
- auparse - Added `MarshallText` method to AuditMessageType to enable the value
  to be marshaled as a string in JSON. faabfa94ec9479bdc1ad6c0334ff178b8193fce5
- aucoalesce - Enhanced aucoalesce to normalize events. 666ff1c30fe624e9fcd9a108b20fceb82331f5fa

### Changed
- Rename RawAuditMessage fields `MessageType` and `RawData` to `Type` and
  `Data` respectively. 8622833714fccd7810669b1265df1c1f918ec0c4
- Make Reassembler concurrency-safe. c57b59c20a684e2a6298a1a5929a79192d76d61b
- auparse - Renamed `address_family` to `family` in parsed sockaddr messages.
  73f97b2f366e6e00acf2ddff4f6575432da3283e

### Deprecated

### Removed

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
  the meaning of these values. #5
- aucoalesce - Added a new package to coalescing related messages into a single
  event. #1

### Changed
- auparse - Changed the behavior of `ParseLogLine()` and `Parse()` to only parse
  the message header. To parse the message body, call `Data()` on the returned
  `AuditMessage`.

### Deprecated

### Removed

## [0.0.1]

### Added
- Added AuditClient for communicating with the Linux Audit Framework in the
  Linux kernel.
- Added auparse package for parsing audit logs.
