# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.1.0]

### Added

- First release of `github.come/elastic/elastic-agent-autodiscover`.
- Add CHANGELOG.md for documenting changes in `elastic-agent-autodiscover`.


[Unreleased]: https://github.com/elastic/elastic-agent-autodiscover/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/elastic/elastic-agent-autodiscover/compare/v0.0.0...v0.1.0


## [0.6.2]

### Changed

- Usage of env var `NODE_NAME` precedes `machine-id` for kubernetes node discovery.


[0.6.2]: https://github.com/elastic/elastic-agent-autodiscover/compare/v0.6.1...v0.6.2


## [0.6.7]

### Changed

- Update NewNodePodUpdater and NewNamespacePodUpdater functions to conditionally check and update kubernetes metadata enrichment of pods


[0.6.7]: https://github.com/elastic/elastic-agent-autodiscover/compare/v0.6.2...v0.6.7

## [0.6.9]

### Changed

- Update GenerateHints function to check supported list of hints


[0.6.9]: https://github.com/elastic/elastic-agent-autodiscover/compare/v0.6.8...v0.6.9

## [0.6.11]

### Changed

- Enhance GenerateHints function to check supported list of hints for multiple datastreams and metricsets


[0.6.10]: https://github.com/elastic/elastic-agent-autodiscover/compare/v0.6.10...v0.6.11
