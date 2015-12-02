# Contributing to Beats

The Beats platform is an open source project and we love to receive
contributions from our community — you!

There are many ways to contribute, from writing tutorials
or blog posts, improving the documentation, submitting bug reports and feature
requests or writing code for implementing a whole new Beat!

Please start by opening a topic on the
[forums](https://discuss.elastic.co/c/beats/libbeat). It may be that somebody
is already working on it, or that there are particular issues that you should
know about before implementing the change.

We enjoy working with contributors to get their code accepted. There are many
approaches to fixing a problem and it is important to find the best approach
before writing too much code. In particular, we are very likely to reject pull
requests that add a new output type (libbeat output for kafka, riemann, etc.).
The reason is that maintaining all these outputs would involve a significant
effort which is already spent in Logstash. You can use Logstash as a gateway
to lots of already supported systems.

The process for contributing to any of the Elastic projects is similar.

## Contribution Steps for libbeat code

1. Test your changes! Run the test suite (`make test`)
2. Please make sure you have signed our [Contributor License
   Agreement](https://www.elastic.co/contributor-agreement/). We are not
   asking you to assign copyright to us, but to give us the right to distribute
   your code without restriction. We ask this of all contributors in order to
   assure our users of the origin and continuing existence of the code. You
   only need to sign the CLA once.
3. Send a pull request! Push your changes to your fork of the repository and
   [submit a pull
   request](https://help.github.com/articles/using-pull-requests). In the pull
   request, describe what your changes do and mention any bugs/issues related
   to the pull request.


## Writing a new Beat

Start in a new repository and use libbeat packages as you would use any other
Go library. Have a look at the way
[Packetbeat](https://github.com/elastic/packetbeat) makes use of these packages
for an example.


## Coding Standards

We try to follow the [Golang coding standards](https://github.com/golang/go/wiki/CodeReviewComments)
 as close as possible. Make sure to run `gofmt` before you push your code.


## Dependency Management

Beats are using [godep](https://github.com/tools/godep) for dependency management.
This means all dependencies are part of the repository. For updating dependencies we
have the following strategy:

* If possible use the most recent release
* If no release tag exist, try to stay as close as possible to master


### Update Dependencies

Godep allows to update all dependencies at once. We DON'T do that. If a dependency
is updated, the newest dependency must be loaded into the `$GOPATH` through either
using

`go get your-go-package-path`

or by having the package already in the `$GOPATH`with the correct version / tag.
To then save the most recent packages into Godep, run

`godep update your-go-package-path`

Avoid using `godep save ./...` or `godep update ...` as this will update all packages at
once and in case of issues it will be hard to track which one cause the issue.

After you updated the package, open a pull request where you state which package
you updated.
