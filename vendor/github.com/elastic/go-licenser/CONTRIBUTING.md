# Contributing

Contributions are very welcome, this includes documentation, tutorials, bug reports, issues, feature requests, feature implementations, pull requests or simply organizing the repository issues.

*Pull requests that contain changes on the code base **and** related documentation, e.g. for a new feature, shall remain a single, atomic one.*

## Building From Source

### Environment Prerequisites

To install the latest changes directly from source code, you will need to have `go` installed with `$GOPATH` defined. If you need assistance with this please follow [golangbootcamp guide](http://www.golangbootcamp.com/book/get_setup#cha-get_setup).

### Actual installation commands

**Make sure you have followed through the environment requisites**

```sh
go get -u github.com/elastic/go-licenser
```

## Reporting Issues

If you have found an issue or defect in `go-licenser` or the latest documentation, use the GitHub [issue tracker](https://github.com/elastic/go-licenser/issues) to report the problem. Make sure to follow the template provided for you to provide all the useful details possible.


### Code Contribution Guidelines

For the benefit of all, here are some recommendations with regards to your PR:

* Go ahead and fork the project and make your changes.  We encourage pull requests to allow for review and discussion of code changes.
* As a best practice it's best to open an Issue on the repository before submitting a PR.
* When you’re ready to create a pull request, be sure to:
    * Sign your commit messages, see [DCO details](https://probot.github.io/apps/dco/)
    * Have test cases for the new code. If you have questions about how to do this, please ask in your pull request.
    * Run `make format` and `make lint`.
    * Ensure that `make unit` succeeds.


### Golden Files

If you're working with a function that relies on testdata or golden files, you might need to update those if your
change is modifying that logic.

```console
$ make update-golden-files
ok  	github.com/elastic/go-licenser	0.029s
```
