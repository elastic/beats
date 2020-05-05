# CHANGELOG
Changelog for ecszap

## 0.2.0 (unreleased)

### Enhancement
* Add `ecszap.ECSCompatibleEncoderConfig` for making existing encoder config ECS conformant [pull#12](https://github.com/elastic/ecs-logging-go-zap/pull/12)
* Add method `ToZapCoreEncoderConfig` to `ecszap.EncoderConfig` for advanced use cases [pull#12](https://github.com/elastic/ecs-logging-go-zap/pull/12)

### Bug Fixes
* Use `zapcore.ISO8601TimeEncoder` as default instead of `ecszap.EpochMicrosTimeEncoder` [pull#12](https://github.com/elastic/ecs-logging-go-zap/pull/12)

### Breaking Change
* remove `ecszap.NewJSONEncoder` [pull#12](https://github.com/elastic/ecs-logging-go-zap/pull/12)

## 0.1.0
Initial Pre-Release supporting [MVP](https://github.com/elastic/ecs-logging/tree/master/spec#minimum-viable-product) for ECS conformant logging 