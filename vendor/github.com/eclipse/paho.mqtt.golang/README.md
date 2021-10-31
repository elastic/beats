
[![PkgGoDev](https://pkg.go.dev/badge/github.com/eclipse/paho.mqtt.golang)](https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang)
[![Go Report Card](https://goreportcard.com/badge/github.com/eclipse/paho.mqtt.golang)](https://goreportcard.com/report/github.com/eclipse/paho.mqtt.golang)

Eclipse Paho MQTT Go client
===========================


This repository contains the source code for the [Eclipse Paho](https://eclipse.org/paho) MQTT 3.1/3.11 Go client library. 

This code builds a library which enable applications to connect to an [MQTT](https://mqtt.org) broker to publish 
messages, and to subscribe to topics and receive published messages.

This library supports a fully asynchronous mode of operation.

A client supporting MQTT V5 is [also available](https://github.com/eclipse/paho.golang).

Installation and Build
----------------------

The process depends upon whether you are using [modules](https://golang.org/ref/mod) (recommended) or `GOPATH`. 

#### Modules

If you are using [modules](https://blog.golang.org/using-go-modules) then `import "github.com/eclipse/paho.mqtt.golang"` 
and start using it. The necessary packages will be download automatically when you run `go build`. 

Note that the latest release will be downloaded and changes may have been made since the release. If you have 
encountered an issue, or wish to try the latest code for another reason, then run 
`go get github.com/eclipse/paho.mqtt.golang@master` to get the latest commit.

#### GOPATH

Installation is as easy as:

```
go get github.com/eclipse/paho.mqtt.golang
```

The client depends on Google's [proxy](https://godoc.org/golang.org/x/net/proxy) package and the 
[websockets](https://godoc.org/github.com/gorilla/websocket) package, also easily installed with the commands:

```
go get github.com/gorilla/websocket
go get golang.org/x/net/proxy
```


Usage and API
-------------

Detailed API documentation is available by using to godoc tool, or can be browsed online
using the [pkg.go.dev](https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang) service.

Samples are available in the `cmd` directory for reference.

Note:

The library also supports using MQTT over websockets by using the `ws://` (unsecure) or `wss://` (secure) prefix in the
URI. If the client is running behind a corporate http/https proxy then the following environment variables `HTTP_PROXY`,
`HTTPS_PROXY` and `NO_PROXY` are taken into account when establishing the connection.

Troubleshooting
---------------

If you are new to MQTT and your application is not working as expected reviewing the
[MQTT specification](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html), which this library implements,
is a good first step. [MQTT.org](https://mqtt.org) has some [good resources](https://mqtt.org/getting-started/) that answer many 
common questions.

### Error Handling

The asynchronous nature of this library makes it easy to forget to check for errors. Consider using a go routine to 
log these: 

```go
t := client.Publish("topic", qos, retained, msg)
go func() {
    _ = t.Wait() // Can also use '<-t.Done()' in releases > 1.2.0
    if t.Error() != nil {
        log.Error(t.Error()) // Use your preferred logging technique (or just fmt.Printf)
    }
}()
```

### Logging

If you are encountering issues then enabling logging, both within this library and on your broker, is a good way to
begin troubleshooting. This library can produce various levels of log by assigning the logging endpoints, ERROR, 
CRITICAL, WARN and DEBUG. For example:

```go
func main() {
	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	// Connect, Subscribe, Publish etc..
}
```

### Common Problems

* Seemingly random disconnections may be caused by another client connecting to the broker with the same client 
identifier; this is as per the [spec](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html#_Toc384800405).
* Unless ordered delivery of messages is essential (and you have configured your broker to support this e.g. 
  `max_inflight_messages=1` in mosquitto) then set `ClientOptions.SetOrderMatters(false)`. Doing so will avoid the 
  below issue (deadlocks due to blocking message handlers).
* A `MessageHandler` (called when a new message is received) must not block (unless 
  `ClientOptions.SetOrderMatters(false)` set). If you wish to perform a long-running task, or publish a message, then 
  please use a go routine (blocking in the handler is a common cause of unexpected `pingresp 
not received, disconnecting` errors). 
* When QOS1+ subscriptions have been created previously and you connect with `CleanSession` set to false it is possible that the broker will deliver retained 
messages before `Subscribe` can be called. To process these messages either configure a handler with `AddRoute` or
set a `DefaultPublishHandler`.
* Loss of network connectivity may not be detected immediately. If this is an issue then consider setting 
`ClientOptions.KeepAlive` (sends regular messages to check the link is active). 
* Brokers offer many configuration options; some settings may lead to unexpected results. If using Mosquitto check
`max_inflight_messages`, `max_queued_messages`, `persistence` (the defaults may not be what you expect).

Reporting bugs
--------------

Please report bugs by raising issues for this project in github https://github.com/eclipse/paho.mqtt.golang/issues

*A limited number of contributors monitor the issues section so if you have a general question please consider the 
resources in the [more information](#more-information) section (your question will be seen by more people, and you are 
likely to receive an answer more quickly).*

We welcome bug reports, but it is important they are actionable. A significant percentage of issues reported are not 
resolved due to a lack of information. If we cannot replicate the problem then it is unlikely we will be able to fix it. 
The information required will vary from issue to issue but consider including:  

* Which version of the package you are using (tag or commit - this should be in your go.mod file)
* A [Minimal, Reproducible Example](https://stackoverflow.com/help/minimal-reproducible-example). Providing an example 
is the best way to demonstrate the issue you are facing; it is important this includes all relevant information
(including broker configuration). Docker (see `cmd/docker`) makes it relatively simple to provide a working end-to-end 
example.
* A full, clear, description of the problem (detail what you are expecting vs what actually happens).
* Details of your attempts to resolve the issue (what have you tried, what worked, what did not).
* [Application Logs](#logging) covering the period the issue occurred. Unless you have isolated the root cause of the issue please include a link to a full log (including data from well before the problem arose).
* Broker Logs covering the period the issue occurred.

It is important to remember that this library does not stand alone; it communicates with a broker and any issues you are 
seeing may be due to:

* Bugs in your code.
* Bugs in this library.
* The broker configuration.
* Bugs in the broker.
* Issues with whatever you are communicating with.

When submitting an issue, please ensure that you provide sufficient details to enable us to eliminate causes outside of
this library.

Contributing
------------

We welcome pull requests but before your contribution can be accepted by the project, you need to create and 
electronically sign the Eclipse Contributor Agreement (ECA) and sign off on the Eclipse Foundation Certificate of Origin. 

More information is available in the 
[Eclipse Development Resources](http://wiki.eclipse.org/Development_Resources/Contributing_via_Git); please take special 
note of the requirement that the commit record contain a "Signed-off-by" entry.

More information
----------------

Discussion of the Paho clients takes place on the [Eclipse paho-dev mailing list](https://dev.eclipse.org/mailman/listinfo/paho-dev).

General questions about the MQTT protocol are discussed in the [MQTT Google Group](https://groups.google.com/forum/?hl=en-US&fromgroups#!forum/mqtt).

There is much more information available via the [MQTT community site](http://mqtt.org).

[Stack Overflow](https://stackoverflow.com/questions/tagged/mqtt+go) has a range questions covering a range of common 
issues (both relating to use of this library and MQTT in general).
