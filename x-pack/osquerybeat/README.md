# {Beat}

Welcome to {Beat}.

Ensure that this folder is at the following location:
`${GOPATH}/src/github.com/elastic/beats/v7/x-pack/osquerybeat`

## Getting Started with {Beat}

### Requirements

* [Golang](https://golang.org/dl/) 1.7

### Init Project
To get running with {Beat} and also install the
dependencies, run the following command:

```
make update
```

It will create a clean git history for each major step. Note that you can always rewrite the history if you wish before pushing your changes.

To push {Beat} in the git repository, run the following commands:

```
git remote set-url origin https://github.com/elastic/beats/v7/x-pack/osquerybeat
git push origin master
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for {Beat} run the command below. This will generate a binary
in the same directory with the name osquerybeat.

```
mage build
```


### Run

To run {Beat} with debugging output enabled, run:

```
./osquerybeat -c osquerybeat.yml -e -d "*"
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

For osquery-extension generated assets (tables/views/docs/README and jumplists lookup maps), use:

```
mage generate
```

By default, jumplists generation uses local cached source files under
`ext/osquery-extension/pkg/jumplists/generate/sources/` for deterministic output.
To refresh those sources on demand:

```
JUMPLISTS_REFRESH_SOURCES=true mage generate
```

### Upgrade the bundled Osquery version

The bundled release metadata is in `internal/distro/distro.json`. To upgrade it:

```bash
mage updateOsquery
```

The target selects the latest stable GitHub release, updates the version and
official artifact checksums, and creates a changelog fragment. Set
`OSQUERY_VERSION` to select a specific release:

```bash
OSQUERY_VERSION=5.23.1 mage updateOsquery
```

The target updates the Osquery version and all artifact checksums used by the
branch.

Review the generated fragment description, then verify the result:

```bash
go test ./internal/distro ./scripts/mage ./scripts/update_osquery
mage fetchOsquerydForTesting
```

This verifies the current host artifact. The packaging jobs download and
verify the complete supported platform matrix.

Then run `mage updateOsqueryManager` in the integrations repository. Set
`BEATS_PATH` to this checkout when the integrations schemas need unreleased
osquerybeat extension specs.

### Cleanup

To clean  {Beat} source code, run the following command:

```
make fmt
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone {Beat} from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/src/github.com/elastic/osquerybeat
git clone https://github.com/elastic/osquerybeat ${GOPATH}/src/github.com/elastic/osquerybeat
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Packaging

The beat frameworks provides tools to crosscompile and package your beat for different platforms. This requires [docker](https://www.docker.com/) and vendoring as described above. To build packages of your beat, run the following command:

```
make release
```

This will fetch and create all images required for the build process. The whole process to finish can take several minutes.
