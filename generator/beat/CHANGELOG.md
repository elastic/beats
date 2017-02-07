# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [Unreleased](...HEAD)
### Added
- Add dev-tools/packer to package the beat for all supported platforms

### Changed
- Use ucfg.Unpack() instead of cfgfile.Read() in Beater.Config method.
- Rename `Configuration` variable in beat struct to `beatConfig` as generalization from @buehler.
- Update Golang dependency to 1.6.0
- Make testing dependent on local version of beats

### Backward Compatibility Breaks
- Remove dependency on cookiecutter

- Renamed BEAT_DIR to BEAT_PATH.
- Renamed BEATNAME to BEAT_NAME.
- Community beats now required BEAT_URL for packaging.

### Bugfixes

### Deprecated
