# {{cookiecutter.beat|capitalize}}

 Welcome to {{cookiecutter.beat|capitalize}}.

Ensure that this folder is at the following location:
`${GOPATH}/{{cookiecutter.beat_path}}`

## To get running with {{cookiecutter.beat|capitalize}}, run the following commands:

```
glide init
glide update --no-recursive
make update
make
```


## To generate etc/{{cookiecutter.beat}}.template.json and etc/{{cookiecutter.beat}}.asciidoc

```
make generate
```

## To run {{cookiecutter.beat|capitalize}} with debugging output enabled, run:

```
./{{cookiecutter.beat}} -c etc/{{cookiecutter.beat}}.yml -e -d "*"
```

## To test {{cookiecutter.beat|capitalize}}, run the following commands:

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

## To clean  {{cookiecutter.beat|capitalize}} source code, run the following commands:

```
make fmt
make simplify
```

## To package {{cookiecutter.beat|capitalize}} for all platforms, run the following commands:

```
cd packer
make
```


## To push {{cookiecutter.beat|capitalize}} in the git repository, run the following commands:

```
git init
git add .
git commit
git remote set-url origin https://{{cookiecutter.beat_path}}/{{cookiecutter.beat}}
git push origin master
```

## To clone {{cookiecutter.beat|capitalize}} from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/{{cookiecutter.beat_path}}
cd ${GOPATH}/{{cookiecutter.beat_path}}
git clone https://{{cookiecutter.beat_path}}/{{cookiecutter.beat}}
```


## For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).
