# Heartbeat (Experimental)

Welcome to Heartbeat.

This is a new EXPERIMENTAL beat for testing service availability using PING based on ICMP, TCP or higher level protocols.

Ensure that this folder is at the following location:
`${GOPATH}/src/github.com/elastic/beats`

## Getting Started with Heartbeat

### Requirements

* [Golang](https://golang.org/dl/) 1.7

### Build

To build the binary for Heartbeat run the command below. This will generate a binary
in the same directory with the name heartbeat.

```
make
```


### Run

To run Heartbeat with debugging output enabled, run:

```
./heartbeat -c heartbeat.yml -e -d "*"
```


### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `fields.yml`.

```
make update
```


### Cleanup

To clean  Heartbeat source code, run the following commands:

```
make fmt
make simplify
```

To clean up the build directory and generated artifacts, run:

```
make clean
```
