---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/beats-contributing.html
---

# Contribute to Beats [beats-contributing]

If you have a bugfix or new feature that you would like to contribute, please start by opening a topic on the [forums](https://discuss.elastic.co/c/beats). It may be that somebody is already working on it, or that there are particular issues that you should know about before implementing the change.

We enjoy working with contributors to get their code accepted. There are many approaches to fixing a problem and it is important to find the best approach before writing too much code. After committing your code, check out the [Elastic Contributor Program](https://www.elastic.co/community/contributor) where you can earn points and rewards for your contributions.

The process for contributing to any of the Elastic repositories is similar.


## Contribution Steps [contribution-steps]

1. Please make sure you have signed our [Contributor License Agreement](https://www.elastic.co/contributor-agreement/). We are not asking you to assign copyright to us, but to give us the right to distribute your code without restriction. We ask this of all contributors in order to assure our users of the origin and continuing existence of the code. You only need to sign the CLA once.
2. Send a pull request! Push your changes to your fork of the repository and [submit a pull request](https://help.github.com/articles/using-pull-requests) using our [pull request guidelines](/extend/pr-review.md). New PRs go to the main branch. The Beats core team will backport your PR if it is necessary.

In the pull request, describe what your changes do and mention any bugs/issues related to the pull request. Please also add a changelog entry to [CHANGELOG.next.asciidoc](https://github.com/elastic/beats/blob/main/CHANGELOG.next.asciidoc).


## Setting Up Your Dev Environment [setting-up-dev-environment]

The Beats are Go programs, so install the 1.22.10 version of [Go](http://golang.org/) which is being used for Beats development.

After [installing Go](https://golang.org/doc/install), set the [GOPATH](https://golang.org/doc/code.md#GOPATH) environment variable to point to your workspace location, and make sure `$GOPATH/bin` is in your PATH.

::::{note}
One deterministic manner to install the proper Go version to work with Beats is to use the [GVM](https://github.com/andrewkroh/gvm) Go version manager. An example for Mac users would be:
::::


```shell
gvm use 1.22.10
eval $(gvm 1.22.10)
```

Then you can clone Beats git repository:

```shell
mkdir -p ${GOPATH}/src/github.com/elastic
git clone https://github.com/elastic/beats ${GOPATH}/src/github.com/elastic/beats
```

::::{note}
If you have multiple go paths, use `${GOPATH%%:*}` instead of `${GOPATH}`.
::::


Beats developers primarily use [Mage](https://github.com/magefile/mage) for development. You can install mage using a make target:

```shell
make mage
```

Then you can compile a particular Beat by using Mage. For example, for Filebeat:

```shell
cd beats/filebeat
mage build
```

You can list all available mage targets with:

```shell
mage -l
```

Some of the Beats might have extra development requirements, in which case you’ll find a CONTRIBUTING.md file in the Beat directory.

We use an [EditorConfig](http://editorconfig.org/) file in the beats repository to standardise how different editors handle whitespace, line endings, and other coding styles in our files. Most popular editors have a [plugin](http://editorconfig.org/#download) for EditorConfig and we strongly recommend that you install it.


## Update scripts [update-scripts]

The Beats use a variety of scripts based on Python, make and mage to generate configuration files and documentation. Ensure to use the version of python listed in the [.python-version](https://github.com/elastic/beats/blob/main/.python-version) file.

The primary command for updating generated files is:

```shell
make update
```

Each Beat has its own `update` target (for both `make` and `mage`), as well as a master `update` in the repository root. If a PR adds or removes a dependency, run `make update` in the root `beats` directory.

Another command properly formats go source files and adds a copyright header:

```shell
make fmt
```

Both of these commands should be run before submitting a PR. You can view all the available make targets with `make help`.

These commands have the following dependencies:

* Python >= 3.7
* Python [venv module](https://docs.python.org/3/library/venv.html)
* [Mage](https://github.com/magefile/mage)

Python venv module is included in the standard library in Python 3. On Debian/Ubuntu systems it also requires to install the `python3-venv` package, that includes additional support scripts:

```shell
sudo apt-get install python3-venv
```


## Selecting Build Targets [build-target-env-vars]

Beats is built using the `make release` target. By default, make will select from a limited number of preset build targets:

* darwin/amd64
* darwin/arm64
* linux/amd64
* windows/amd64

You can change build targets using the `PLATFORMS` environment variable. Targets set with the `PLATFORMS` variable can either be a GOOS value, or a GOOS/arch pair. For example, `linux` and `linux/amd64` are both valid targets. You can select multiple targets, and the `PLATFORMS` list is space delimited, for example `darwin windows` will build on all supported darwin and windows architectures. In addition, you can add or remove from the list of build targets by prepending `+` or `-` to a given target. For example: `+bsd` or `-darwin`.

You can find the complete list of supported build targets with `go tool dist list`.


## Linting [running-linter]

Beats uses [golangci-lint](https://golangci-lint.run/). You can run the pre-configured linter against your change:

```shell
mage llc
```

`llc` stands for `Lint Last Change` which includes all the Go files that were changed in either the last commit (if you’re on the `main` branch) or in a difference between your feature branch and the `main` branch.

It’s expected that sometimes a contributor will be asked to fix linter issues unrelated to their contribution since the linter was introduced later than changes in some of the files.

You can also run the linter against an individual package, for example the filbeat command package:

```shell
golangci-lint run ./filebeat/cmd/...
```


## Testing [running-testsuite]

You can run the whole testsuite with the following command:

```shell
make testsuite
```

Running the testsuite has the following requirements:

* Python >= 3.7
* Docker >= 1.12
* Docker-compose >= 1.11

For more details, refer to the [Testing](/extend/testing.md) guide.


## Documentation [documentation]

All of the Beats documentation is located in the `elastic/beats` repository:

* `beats/docs/extend` contains documentation about developing and contributing to the Beats code.
* `beats/docs/reference` contains the docs for each individual Beat, as well as some content that is common to all Beats in the `beats/docs/reference/libbeat` directory.
* `beats/docs/release-notes` contains all of the product update notes.

Beginning with version 9.0.0, all Elastic documentation is sourced in Markdown format. For general information about contributing to the Elastic documentation, including versioning guidelines, a syntax reference, and more,  refer to the [Elastic Docs v3 welcome page](https://elastic.github.io/docs-builder/).

## Dependencies [dependencies]

In order to create Beats we rely on Golang libraries and other external tools.


### Other dependencies [_other_dependencies]

Besides Go libraries, we are using development tools to generate parsers for inputs and processors.

The following packages are required to run `go generate`:


#### Auditbeat [_auditbeat]

* FlatBuffers >= 1.9


#### Filebeat [_filebeat]

* Graphviz >= 2.43.0
* Ragel >= 6.10


## Changelog [changelog]

To keep up to date with changes to the official Beats for community developers, follow the developer changelog [here](https://github.com/elastic/beats/blob/main/CHANGELOG-developer.next.asciidoc).



