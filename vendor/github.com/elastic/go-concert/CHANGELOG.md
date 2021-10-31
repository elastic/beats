# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.2.0]

### Added

- `(*TaskGroup.Context()` returns the groups internal Context. (#48)
- `(*TaskGroup).Wait()` waits for all managed go routines to return. (#48)
- `TaskGroup.MaxError`configures the maxium number of errors to keep. The
  last N errors will be reported. (#48)
- Introduce support for configurable group stop behavior via `OnQuit`
  callbacks. Provided strategies: `ContinueOnErrors`, `RestartOnError`,
  `StopAll`, `StopOnError`, `StopOnErrorOrCancel`. (#48)

### Changed

- `(*TaskGroup).Go` will now pass `context.Context` to the run function. (#48)
- `TaskGroup.StopOnError` has been superseded by `TaskGroup.OnQuit`. (#48)
- `TaskGroup` will now shutdown by default if a managed go-routine did return an error. (#48)

## [0.1.0]

### Added

- Added SafeWaitGroupWithCancel constructor that registers the SafeWaitGroup
  with a context.Context in order to signal 'Close'. #39
- Added TaskGroupWithCancel constructor that registers the TaskGroup with a
  context.Context in order to signal 'Close'. #39
- Added `CancelContext`, `WithCancelContext`, and `WrapCancel` in order to combine
  a `context.Context` and a `context.CancelFunc` into a common struct that can be
  stored as a single field in a struct. (#41)
- Add `unison.Cell` type. A cell stores states that can be updated
  asynchronously without back-pressure (unlike a channel). Consumers can read the current state
  or wait for new updates. (#44)

### Changed

- Many helpers in ctxtool now accept a simplified interface that only
  implements `Done()` and `Err()`, to simplify use if ctxtool is used to
  integrate with other types then context.Context. If the input type is already
  context.Context, its type and functionality (Value, Deadline) are preserved.
  #39

### Removed

- Remove `chorus` package with types `CloseRef` and `Closer` (#40)
- Remove `OnceSignaler` (#42)
- Remove `unison.LockManager` (#42)

## [0.0.4]

### Added

- Add `osctx.WithSignal` utility function, which creates a context that will be cancelled if the process receives an os.Signal. (#31)

### Changed

- Update `timed.Period` to return an error. The callback is required to return an error as well. (#28)

### Fixed

- Fix TaskGroup reporting context.Cancel as error. (#30)

## [0.0.3]

### Added
- Add timed.Wait and timed.Periodic helpers for using timed operations with context.Context based cancellation. (#26)

## [0.0.2]

### Added

- Add MultiErrGroup (#20).
- Add Group interface and TaskGroup implementation (#23).
- Add SafeWaitGroup (#23).
- Add ClosedGroup (#24).

### Changed

- FromCancel returns original context.Context, if input implements this type. Deadline and Value will not be ignored anymore. (#22)


[Unreleased]: https://github.com/elastic/go-concert/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/elastic/go-concert/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/elastic/go-concert/compare/v0.0.4...v0.1.0
[0.0.4]: https://github.com/elastic/go-concert/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/elastic/go-concert/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/elastic/go-concert/compare/v0.0.1...v0.0.2
