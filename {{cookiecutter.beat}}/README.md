# {{cookiecutter.beat|capitalize}}

Welcome to {{cookiecutter.beat|capitalize}}.

Ensure that this folder is at the following location:
`${GOPATH}/{{cookiecutter.beat_path}}`

## Getting Started with {{cookiecutter.beat|capitalize}}

### Init Project
To get running with {{cookiecutter.beat|capitalize}}, run the following commands:

```
make init
```


To push {{cookiecutter.beat|capitalize}} in the git repository, run the following commands:

```
git commit 
git remote set-url origin https://{{cookiecutter.beat_path}}/{{cookiecutter.beat}}
git push origin master
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for {{cookiecutter.beat|capitalize}} run the command below. This will generate a binary
in the same directory with the name {{cookiecutter.beat}}.

```
make
```


### Run

To run {{cookiecutter.beat|capitalize}} with debugging output enabled, run:

```
./{{cookiecutter.beat}} -c {{cookiecutter.beat}}.yml -e -d "*"
```


### Test

To test {{cookiecutter.beat|capitalize}}, run the following commands:

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


### Package

To cross-compile and package {{cookiecutter.beat|capitalize}} for all supported platforms, run the following commands:

```
cd dev-tools/packer
make deps
make images
make
```

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `etc/fields.yml`.
To generate etc/{{cookiecutter.beat}}.template.json and etc/{{cookiecutter.beat}}.asciidoc

```
make update
```


### Cleanup

To clean  {{cookiecutter.beat|capitalize}} source code, run the following commands:

```
make fmt
make simplify
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone {{cookiecutter.beat|capitalize}} from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/{{cookiecutter.beat_path}}
cd ${GOPATH}/{{cookiecutter.beat_path}}
git clone https://{{cookiecutter.beat_path}}/{{cookiecutter.beat}}
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).
