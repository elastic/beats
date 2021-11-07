# Elastic Common Schema (ECS) support for uber-go/zap logger

Use this library for automatically adding a minimal set of ECS fields to your logs, when using [uber-go/zap](https://github.com/uber-go/zap).
 
---

**Please note** that this library is in a **beta** version and backwards-incompatible changes might be introduced in future releases. While we strive to comply to [semver](https://semver.org/), we can not guarantee to avoid breaking changes in minor releases.

---
 
The encoder logs in JSON format, relying on the default [zapcore/json_encoder](https://github.com/uber-go/zap/blob/master/zapcore/json_encoder.go) when possible. 

Following fields will be added by default:
```
{
    "log.level":"info",
    "@timestamp":1583748236254129,
    "message":"some logging info",
    "ecs.version":"1.6.0"
}
```

It also takes care of logging error fields in [ECS error format](https://www.elastic.co/guide/en/ecs/current/ecs-error.html). 

## What is ECS?

Elastic Common Schema (ECS) defines a common set of fields for ingesting data into Elasticsearch.
For more information about ECS, visit the [ECS Reference Documentation](https://www.elastic.co/guide/en/ecs/current/ecs-reference.html).

## Installation
Add the package to your `go.mod` file
```
require go.elastic.co/ecszap master
```

## Example usage
### Set up default logger
```go
encoderConfig := ecszap.NewDefaultEncoderConfig()
core := ecszap.NewCore(encoderConfig, os.Stdout, zap.DebugLevel)
logger := zap.New(core, zap.AddCaller())
```

### Use structured logging
```go
// Adding fields and a logger name
logger = logger.With(zap.String("custom", "foo"))
logger = logger.Named("mylogger")

// Use strongly typed Field values
logger.Info("some logging info",
    zap.Int("count", 17),
    zap.Error(errors.New("boom")),
}

	// Log Output:
	//{
	//	"log.level":"info",
	//	"@timestamp":1584716847523456,
	//	"log.logger":"mylogger",
	//	"log.origin":{
	//		"file.name":"main/main.go",
	//		"file.line":265
	//	},
	//	"message":"some logging info",
	//	"ecs.version":"1.6.0",
	//	"custom":"foo",
	//	"count":17,
	//	"error":{
	//		"message":"boom"
	//	}
	//}
```

### Log errors
```go
err := errors.New("boom")
logger.Error("some error", zap.Error(pkgerrors.Wrap(err, "crash")))

	// Log Output:
	//{
	//	"log.level":"error",
	//	"@timestamp":1584716847523842,
	//	"log.logger":"mylogger",
	//	"log.origin":{
	//		"file.name":"main/main.go",
	//		"file.line":290
	//	},
	//	"message":"some error",
	//	"ecs.version":"1.6.0",
	//	"custom":"foo",
	//	"error":{
	//		"message":"crash: boom",
	//		"stacktrace": "\nexample.example\n\t/Users/xyz/example/example.go:50\nruntime.example\n\t/Users/xyz/.gvm/versions/go1.13.8.darwin.amd64/src/runtime/proc.go:203\nruntime.goexit\n\t/Users/xyz/.gvm/versions/go1.13.8.darwin.amd64/src/runtime/asm_amd64.s:1357"
	//	}
	//}
```

### Use sugar logger
```go
sugar := logger.Sugar()
sugar.Infow("some logging info",
    "foo", "bar",
    "count", 17,
)

	// Log Output:
	//{
	//	"log.level":"info",
	//	"@timestamp":1584716847523941,
	//	"log.logger":"mylogger",
	//	"log.origin":{
	//		"file.name":"main/main.go",
	//		"file.line":311
	//	},
	//	"message":"some logging info",
	//	"ecs.version":"1.6.0",
	//	"custom":"foo",
	//	"foo":"bar",
	//	"count":17
	//}
```

### Wrap a custom underlying zapcore.Core
```go
encoderConfig := ecszap.NewDefaultEncoderConfig()
encoder := zapcore.NewJSONEncoder(encoderConfig.ToZapCoreEncoderConfig())
syslogCore := newSyslogCore(encoder, level) //create your own loggers
core := ecszap.WrapCore(syslogCore)
logger := zap.New(core, zap.AddCaller())
```

### Transition from existing configurations
```go
encoderConfig := ecszap.ECSCompatibleEncoderConfig(zap.NewDevelopmentEncoderConfig())
encoder := zapcore.NewJSONEncoder(encoderConfig)
core := zapcore.NewCore(encoder, os.Stdout, zap.DebugLevel)
logger := zap.New(ecszap.WrapCore(core), zap.AddCaller())
```

## References
* Introduction to ECS [blog post](https://www.elastic.co/blog/introducing-the-elastic-common-schema).
* Logs UI [blog post](https://www.elastic.co/blog/infrastructure-and-logs-ui-new-ways-for-ops-to-interact-with-elasticsearch).

## Test
```
go test ./...
```

## Contribute
Create a Pull Request from your own fork. 

Run `mage` to update and format you changes before submitting. 

Add new dependencies to the NOTICE.txt.

## License
This software is licensed under the [Apache 2 license](https://github.com/elastic/ecs-logging-go/zap/blob/master/LICENSE). 
