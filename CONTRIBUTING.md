# Contributing to Packetbeat

Packetbeat is an open source project and we love to receive contributions from
our community — you! There are many ways to contribute, from writing tutorials
or blog posts, improving the documentation, submitting bug reports and feature
requests or writing code for implementing a whole new protocol.

If you have a bugfix or new feature that you would like to contribute to
Packetbeat, please start by opening a topic on the
[forums](https://discuss.elastic.co/c/beats/packetbeat). It may be that
somebody is already working on it, or that there are particular issues that you
should know about before implementing the change.

We enjoy working with contributors to get their code accepted. There are many
approaches to fixing a problem and it is important to find the best approach
before writing too much code.

The process for contributing to any of the Elastic repositories is similar.

## Contribution Steps

1. Test your changes! Run the test suite ('make test')
2. Please make sure you have signed our [Contributor License
   Agreement](http://www.elasticsearch.org/contributor-agreement/). We are not
   asking you to assign copyright to us, but to give us the right to distribute
   your code without restriction. We ask this of all contributors in order to
   assure our users of the origin and continuing existence of the code. You
   only need to sign the CLA once.
3. Send a pull request! Push your changes to your fork of the repository and
   [submit a pull
   request](https://help.github.com/articles/using-pull-requests). In the pull
   request, describe what your changes do and mention any bugs/issues related
   to the pull request.


## Compiling Packetbeat

Packetbeat is a Go program, so install [golang](http://golang.org/) if you
don't have it already. The only other mandatory dependency is `libpcap` which
often is pre-installed on your operating system.

The location where you clone is important. Please clone under the source
directory of your `GOPATH`. If you don't have `GOPATH` already set, you can
simply set it to your home directory (`export GOPATH=$HOME`).

    $ mkdir -p $GOPATH/src/github.com/elastic
    $ cd $GOPATH/src/github.com/elastic
    $ git clone https://github.com/elastic/packetbeat.git

To build Packetbeat successfully, first you need to get all the Go
dependencies:

    $ cd packetbeat
    $ make deps

and then compile it with:

    $ make
