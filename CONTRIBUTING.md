Please post all questions and issues first on
[https://discuss.elastic.co/c/beats](https://discuss.elastic.co/c/beats)
before opening a Github Issue.

# Contributing to Beats

The Beats are open source and we love to receive contributions from our
community — you!

There are many ways to contribute, from writing tutorials or blog posts,
improving the documentation, submitting bug reports and feature requests or
writing code for implementing a whole new protocol.

If you have a bugfix or new feature that you would like to contribute, please
start by opening a topic on the [forums](https://discuss.elastic.co/c/beats).
It may be that somebody is already working on it, or that there are particular
issues that you should know about before implementing the change.

We enjoy working with contributors to get their code accepted. There are many
approaches to fixing a problem and it is important to find the best approach
before writing too much code.

The process for contributing to any of the Elastic repositories is similar.

## Contribution Steps

1. Please make sure you have signed our [Contributor License
   Agreement](https://www.elastic.co/contributor-agreement/). We are not
   asking you to assign copyright to us, but to give us the right to distribute
   your code without restriction. We ask this of all contributors in order to
   assure our users of the origin and continuing existence of the code. You
   only need to sign the CLA once.
2. Send a pull request! Push your changes to your fork of the repository and
   [submit a pull
   request](https://help.github.com/articles/using-pull-requests). In the pull
   request, describe what your changes do and mention any bugs/issues related
   to the pull request.


## Adding a new Beat

If you want to create a new Beat, please read our [developer
guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).
You don't need to submit the code to this repository. Most new Beats start in
their own repository and just make use of the libbeat packages. After you have
a working Beat that you'd like to share with others, open a PR to add it to our
list of [community
Beats](https://github.com/elastic/beats/blob/master/libbeat/docs/communitybeats.asciidoc).

## Setting up your dev environment

The Beats are Go programs, so install the latest version of
[golang](http://golang.org/) if you don't have it already. The current Go version
used for development is Golang 1.7.

The location where you clone is important. Please clone under the source
directory of your `GOPATH`. If you don't have `GOPATH` already set, you can
simply set it to your home directory (`export GOPATH=$HOME`).

    $ mkdir -p $GOPATH/src/github.com/elastic
    $ cd $GOPATH/src/github.com/elastic
    $ git clone https://github.com/elastic/beats.git

Then you can compile a particular Beat by using the Makefile. For example, for
Packetbeat:

    $ cd beats/packetbeat
    $ make

Some of the Beats might have extra development requirements, in which case you'll find a
CONTRIBUTING.md file in the Beat directory.

## Update scripts

The Beats use a variety of scripts based on Python to generate configuration files
and documentation. The command used for this is:

    $ make update

This command has the following dependencies:

* Python >=2.7.9
* [virtualenv](https://virtualenv.pypa.io/en/latest/) for Python

Virtualenv can be installed with the command `easy_install virtualenv` or `pip install virtualenv`.
More details can be found [here](https://virtualenv.pypa.io/en/latest/installation.html).


## Testing

You can run the whole testsuite with the following command:

    $ make testsuite

Running the testsuite has the following requirements:

* Python >=2.7.9
* Docker >=1.10.0
* Docker-compose >= 1.7.0


## Documentation

The documentation for each Beat is located under {beatname}/docs and is based on asciidoc. After changing the docs,
you should verify that the docs are still building to avoid breaking the automated docs build. To build the docs run
`make docs`. If you want to preview the docs for a specific Beat, run `make docs-preview`
inside the folder for the Beat. This will automatically open your browser with the docs for preview.


## Dependencies

To manage the `vendor/` folder we use
[glide](https://github.com/Masterminds/glide), which uses
[glide.yaml](glide.yaml) as a manifest file for the dependencies. Please see
the glide documentation on how to add or update vendored dependencies.

