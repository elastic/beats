# ecslog

ecslog is an experimental structured logger for the Go programming language.

**TOC**
- [Structure](#structure)
- [Concepts](#concepts)
  * [Fields](#fields)
  * [Context](#context)
  * [Capturing Format strings](#capturing-format-strings)
  * [Errors](#errors)
- [Use genfields](#use-genfields)

Aim of this project is to create a type safe logger generating log events which
are fully compatible to the [Elastic Common Schema
(ECS)](https://github.com/elastic/ecs). ECS defines a common set of fields for
collecting, processing, and ingesting data within the [Elastic Stack](https://www.elastic.co/guide/en/elastic-stack/current/elastic-stack.html#elastic-stack).

Logs should be available for consumption by developers, operators, and any kind
of automated processing (index for search, store in databases, security
analysis, alerting).

While developers want to add additional state to log messages
troubleshooting, other users might not gain much value from unexplained
internal state being printed. First and foremost logs should be
self-explanatory messages.
Yet in the presence of micro-services and highly multithreaded applications
standardized context information is mandatory for filtering and correlating
relevant log messages by machine, service, thread, API call or user.

Ideally automated processes should not have to deal with parsing the actual
message. Messages can easily change between releases, and should be ignored at
best. We can and should provide as much insight into our logs as possible with
the help of additional meta-data used to annotate the log message.

Using untyped and schemaless structured logging, we put automation at
risk of breaking, or requiring operators to adapt transformation every now and then.
There is always the chance of developers, removing or renaming fields. Or using
the same field names, but with values of different types. Some consequences of
undetected schema changes are:
- A subset of logs might not be indexible in an Elasticsearch Index anymore due
  to mapping conflicts for example.
- Scripts/Applications report errors or crash due to unexpected types
- Analysers produce wrong results due to expected fields becoming missing or new ones have been added.

Creating logs based on a common schema like ECS helps in defining and
guaranteeing a common log structure a many different stakeholders can rely on
(See [What are the benefits of using ECS?](https://github.com/elastic/ecs#what-are-the-benefits-of-using-ecs).).
ECS defines a many common fields, but is still extensible (See
[Fields](https://github.com/elastic/ecs#fields)). ECS defines a core level and
an extended level, [reserves some common
namespaces](https://github.com/elastic/ecs#reserved-section-names). It is not
fully enclosed, but meant to be extended, so to fit an
applications/organizations needs.

ecslog distinguishes between standardized and user(developer) provided fields.
The standardized fields are type-safe, by providing developers with type-safe
field constructors. These are checked at compile time and guarantee that the
correct names will be used when publish the structured log messages.

ECS [defines its schema in yaml files](https://github.com/elastic/ecs/tree/master/schemas).
These files are compatible to `fields.yml` files, that are also used in the Elastic
Beats project. Among others Beats already generate Documentation, Kibana index
patterns, [Elasticsearch Index Templates](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates.html)
based on these definitions.

ecslog reuses the definitions provided by ECS, so to generate the code for the
type-safe ECS compatible field constructors (See [tool
sources](https://github.com/urso/ecslog/tree/master/cmd/genfields)).

Using the available definitions and tools it is possible to create log events,
which are normalized and storable into Elasticsearch as is.


## Structure

**Packages**:
- **.**: Top level package defining the public logger.
- **./backend** logger backend interface definitions and composable implementations for building actual logging outputs.
- **./ctxtree**: internal representation of log and error contexts.
- **./fld**: Support for fields.
- **./fld/ecs**: ECS field constructors.
- **./errx**: Error support package with support for:
  - wrapping/annotating errors with additional context
  - querying errors by predicate, contents, type
  - walking trace/tree of errors

## Concepts

### Fields

ecslog differentiates between standardized fields and user fields. We provide
type safe constructors for standardized fields, but user defined fields are not
necessarily type-safe and often carry additional debug information for
consumption by the actual developer. Consumers of logs should be prepared to
remove user fields from log messages if necessary.

The structured logging backends mix in standardized fields as is, right at the
root of the document/event to be generated. User fields are prefixed by
`fields.`.

This log statement using the standardized `ecs.agent.name` field and the user defined `myfield`:

```
	log.With(
		// ECS standardized field
		ecs.Agent.Name("myapp"),

		// user field
		"myfield", "test",
	).Info("info message")
```

produces this JSON document:

```
    {
      ...
      "agent": {
        "name": "myapp"
      },
      "fields": {
        "myfield": "test"
      },
      "log": {
        ...
      },
      "message": "info message"
    }
```

### Context

The logger it's context is implemented by the **ctxtree** package.
Fields can only be added to an context, but not be removed or updated.

A field added twice to a context will be reported only once, ensuring tools
operating on the log message always receive a well defined JSON document.
Calling: 

```
	log.With("field", 1, "field", 2).Info("hello world")
```

or:

```
	log.With("field", 1).With("field", 2).Info("hello world")
```

produces:

```
    {
      ...
      "fields": {
        "field": 2
      },
      "log": {
        ...
      },
      "message": "hello world"
    }
```

Internally the context is represented as a tree. Within one node in the tree,
fields are ordered by the order they've been added to the context.
When creating a context, one can pass a 'predecessor' and a 'successor' to the
context. A snapshot of the current state of these contexts will be used, so to
allow concurrent use  of contexts.

The order of fields in a context-tree is determined by an depth-first traversal
of all contexts in the tree. This is used to link contexts between loggers 
top-down, while linking contexts of error values from the bottom upwards.

### Capturing Format strings

The logging methods `Tracef`, `Debugf`, `Infof`, `Errorf` require a format
string as first argument. The intend of these methods is to create readable
and explanatory message.

The format strings supported are mostly similar to the fmt.Printf family, but add
support for capturing additional user fields in the current log context:

```
	log.Errorf("Can not open '%{file}'.", "file.txt")
```

produces this document:

```
{
  ...
  "fields": {
    "file": "file.txt"
  },
  "log": {
    ...
  },
  "message": "Can not open 'file.txt'."
}
```


Applications should log messages like `"can not open file.txt"` instead of
`"can not open file"` forcing the user to look at configuration or additional
fields in the log message. This is amplified by the fact that ecslog 
backends can supress the generation of the context when logging. The text backend
without context capturing will just print:

```
2019-01-05T20:30:25+01:00 ERROR	main.go:79	can not open file.txt
```

Standardized fields can also be passed to a format string via:

```
	log.Errorf("Failed to access %v", ecs.File.Path("test.txt"))
```

or:

```
	log.Errorf("Failed to access '%{file}'", ecs.File.Path("test.txt"))
```

Both calls produce the document:

```
{
  ...
  "file": {
    "path": "test.txt"
  },
  "log": {
    ...
  },
  "message": "Failed to access 'test.txt'"
}
```

### Errors

Error values serve multiple purposes. Error values are not
only used to signal an error to the caller, but also give the programmer a
chance to act on errors by interrogating the error value. Eventually an error
value is logged as well for troubleshooting. In presence of structured logging 
an error value should support:
- Examining the value by the source code.
- Create self-explanatory human readable message.
- Carry additional context for automated processes consuming logs with errors (e.g. alerting).
- Serialize/Print/Examine causes of the error as well.

Error values tend to be passed bottom-up from the root cause(s), until they are
eventually logged. So to understand the root cause and the actual context in
which the error was produced it is a good idea to annotate errors with
additional context while bubbling up. 

By properly annotating/wrapping errors we end up with a call-trace.  ecslog
assumes the trace to be a tree, so to also capture and represent multi-error
values. The root-cause(s) are the leaf-node(s) in the tree.

Packages often used for wrapping/annotating errors are:
- github.com/hashicorp/errwrap
- github.com/pkg/errors
- github.com/hashicorp/go-multierror
- go.uber.org/multierr
- github.com/joeshaw/multierror

Difficulty with the many error packages is consistent handling and logging of
errors. For example different means on accessing an errors cause. The error
interface mandates an error implementing `Error() string` only. Some packages
also implement the `fmt.Formatter` interface, so to only print the full trace
if the format string `'%+v'` is used. This easily leads to confusion on how to
log an error, potentially not logging the actual root cause.

For getting some consistency when dealing with error values `ecslog/errx`
provides utility functions for wrapping, annotating, and examining error
values.

Functions for iterating all errors in an error tree are: `Iter`, `Walk`, `WalkEach`

For manual walking an error tree one can use `NumCauses` and `Cause`.

The errx can examine the error trees of error types implementing `Cause()
error`, `WrapperErrors() []error`, and `NumCauses() int, Cause(i int) error`.
This makes it compatible to a number of custom error packages, but not all.

`errx` also provides `ContainsX/FindX/CollectX` functions. These support custom
predicates, types, or sentinal error values.

We can also use `errx` to wrap errors via `Errf`, `Wrap`, `WrapAll`. All these
functions support Capturing Format strings, so to add additional context for
logging. The location in the source code will also be captured when using the
error constructors/wrappers.

For example:
```
	errx.Wrap(io.ErrUnexpectedEOF, "failed to read %{file}", "file.txt")
->
  {
    {
      "file": ".../main.go",
      "line": 128
    },
    "cause": {
      "message": "unexpected EOF"
    },
    "ctx": {
      "fields": {
        "file": "file.txt"
      }
    },
    "message": "failed to read file.txt: unexpected EOF"
  }
```

We can add some additional context for logging via `errx.With`:

```
	errx.With(
		ecs.HTTP.Request.Method("GET"),
		ecs.URL.Path("/get_file/file.txt"),
	).Wrap(io.ErrUnexpectedEOF, "failed to read %{file}", "file.txt")
->
  {
    "at": {
      "file": ".../main.go",
      "line": 46
    },
    "cause": {
      "message": "unexpected EOF"
    },
    "ctx": {
      "fields": {
        "file": "file.txt"
      },
      "http": {
        "request": {
          "method": "GET"
        }
      },
      "url": {
        "path": "/get_file/file.txt"
      }
    },
    "message": "failed to read file.txt: unexpected EOF"
  }
```

The logger backends rely on `errx` for examining and serializing errors in a
consistent way (best effort).

When serializing errors, the combined context is added to the `ctx` field.
The 'local' error message (as reporter via `Error() string`) is added to the
`message` field.

The location will be added if the error value implements `At() (string, int)`.
Multi-cause errors will add an array with each error value to the `causes` field.


Example:
```
	seviceLog := log.With(
		ecs.Service.Name("my server"),
		ecs.Host.Hostname("localhost"),
	)

	...

	handlerLog := seviceLog.With(
		ecs.HTTP.Request.Method("GET"),
		ecs.URL.Path("/get_file/file.txt"),
		ecs.Source.Domain("localhost"),
		ecs.Source.IP("127.0.0.1"),
	)

	... 

	file := "file.txt"

	err := errx.With(
		ecs.File.Path(file),
		ecs.File.Extension("txt"),
		ecs.File.Owner("me"),
	).Wrap(io.ErrUnexpectedEOF, "failed to read file")

	...

	handlerLog.Error("Failed to serve %v: %v", ecs.File.Path(file), err)
```

JSON log message:

```
{
  "@timestamp": "2019-01-05T20:16:04.865708+01:00",
  "error": {
    "at": {
      "file": ".../main.go",
      "line": 46
    },
    "cause": {
      "message": "unexpected EOF"
    },
    "ctx": {
      "file": {
        "extension": "txt",
        "owner": "me",
        "path": "file.txt"
      }
    },
    "message": "failed to read file: unexpected EOF"
  },
  "fields": {
    "custom": "value",
    "nested": {
      "custom": "another value"
    }
  },
  "file": {
    "path": "file.txt"
  },
  "host": {
    "hostname": "localhost"
  },
  "http": {
    "request": {
      "method": "GET"
    }
  },
  "log": {
    "file": {
      "basename": "main.go",
      "line": 154,
      "path": ".../ecslog/cmd/tstlog/main.go"
    },
    "level": "error"
  },
  "message": "Failed to serve file.txt: failed to read file: unexpected EOF",
  "service": {
    "name": "my server"
  },
  "source": {
    "domain": "localhost",
    "ip": "127.0.0.1"
  },
  "url": {
    "path": "/get_file/file.txt"
  }
}
```


## Use genfields

The genfields script (found in cmd/genfields) should be used to convert an ECS
compatible schema definition to type safe field constructors that can be used
in Go code.

The fld/ecs/0gen.go source file uses genfields for creating the ECS field constructors via
go generate.

The genfields parses the schema definition from a directory containing the
schema yaml files: 

```
genfields -out schema.go -fmt -schema <path to schema>
```

