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
used for development is Golang 1.5.2.

The Beats are Go programs, so install the latest version of
[golang](http://golang.org/) if you don't have it already.

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

Some of the Beats might have extra development requirements, in which case a
CONTRIBUTING.md file is find in the Beat directory.

You can run the whole testsuite with the following command:

    # make testsuite

## Dependencies

The Beats project is using the [Go 1.5 vendor
experiment](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo/edit)
for its dependencies. This means the Go dependencies code is copied under the
`vendor/` directory and committed into source control.

To manage the `vendor/` folder we use
[glide](https://github.com/Masterminds/glide), which uses
[glide.yaml](glide.yaml) as a manifest file for the dependencies. Please see
the glide documentation on how to add or update vendored dependencies.

