# {Beat}

Welcome to {Beat}.

Ensure that this folder is at the following location:
`${GOPATH}/src/{beat_path}`

## Getting Started with {Beat}

### Requirements

* [Golang](https://golang.org/dl/) 1.9.2

### Init Project
To get running with {Beat} and also install the
dependencies, run the following command:

```
make setup
```

To push {Beat} in the git repository, run the following commands:

```
git remote set-url origin https://{beat_path}
git commit -m "Initial commit"
git push origin master
```

By default, the {Beat} version is the same as libbeats.
To specify the {Beat} version, run the following commands:

```
make set_version VERSION=1.2.3
make update
make
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for {Beat} run the command below. This will generate a binary
in the same directory with the name {beat}.

```
make
```


### Run

To run {Beat} with debugging output enabled, run:

```
./{beat} -c {beat}.yml -e -d "*"
```


### Test

To test {Beat}, run the following command:

```
make testsuite
```

alternatively:
```
make unit-tests
make system-tests
make integration-tests
make coverage-report
```

The test coverage is reported in the folder `./build/coverage/`

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `fields.yml` by running the following command.

```
make update
```


### Cleanup

To clean  {Beat} source code, run the following commands:

```
make fmt
make simplify
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone {Beat} from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/src/{beat_path}
git clone https://{beat_path} ${GOPATH}/src/{beat_path}
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Packaging

The beat frameworks provides tools to crosscompile and package your beat for different platforms. This requires [docker](https://www.docker.com/) and vendoring as described above. To build packages of your beat, run the following command:

```
make package
```

This will fetch and create all images required for the build process. The whole process to finish can take several minutes.
