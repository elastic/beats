# Contributing to the Go APM Agent

The Go APM Agent is open source and we love to receive contributions from our community — you!

There are many ways to contribute, from writing tutorials or blog posts, improving the
documentation, submitting bug reports and feature requests or writing code.

You can get in touch with us through [Discuss](https://discuss.elastic.co/c/apm).
Feedback and ideas are always welcome.

## Code contributions

If you have a bugfix or new feature that involves significant changes that you would like to
contribute, please find or open an issue to discuss the changes first. It may be that somebody
is already working on it, or that there are particular issues that you should know about before
implementing the change.

For minor changes (e.g. fixing a typo), you can just send your changes.

### Submitting your changes

Generally, we require that you test any code you are adding or modifying. Once your changes are
ready to submit for review:

1. Sign the Contributor License Agreement

    Please make sure you have signed our [Contributor License Agreement](https://www.elastic.co/contributor-agreement/).
    We are not asking you to assign copyright to us, but to give us the right to distribute
    your code without restriction. We ask this of all contributors in order to assure our
    users of the origin and continuing existence of the code. You only need to sign the CLA once.

2. Test your changes

    Run the test suite to make sure that nothing is broken.
    See [testing](#testing) for details.

3. Review your changes

    Before sending your changes for review, it pays to review it yourself first!

    If you're making significant changes, please familiarize yourself with [Effective Go](https://golang.org/doc/effective_go.html)
    and [go/wiki/CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments).
    These documents will walk you through writing idiomatic Go code, which we strive for.

    Here are a few things to check:
    - format the code with [gofmt](https://golang.org/cmd/gofmt/) or [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports)
    - lint your code using [golint](https://github.com/golang/lint)
    - check for common errors using [go vet](https://golang.org/cmd/vet/)

4. Rebase your changes

    Update your local repository with the most recent code from the main repo, and rebase your
    branch on top of the latest master branch.  We prefer your initial changes to be squashed
    into a single commit. Later, if we ask you to make changes, add them as separate commits.
    This makes them easier to review. As a final step before merging we will either ask you to
    squash all commits yourself or we'll do it for you.

5. Submit a pull request

    Push your local changes to your forked copy of the repository and [submit a pull request](https://help.github.com/articles/using-pull-requests).
    In the pull request, choose a title which sums up the changes that you have made, and in
    the body provide more details about what your changes do, and the reason for making them.
    Also mention the number of the issue where discussion has taken place, or issues that are
    fixed/closed by the changes, e.g. "Closes #123".

6. Be patient

    We might not be able to review your code as fast as we would like to, but we'll do our
    best to dedicate it the attention it deserves. Your effort is much appreciated!

### Testing

The tests currently do not require any external resources, so just run `go test ./...`.
We test with all versions of Go from 1.8 onwards using [Travis CI](https://travis-ci.org).

We track code coverage. 100% coverage is not a goal, but please do check that your tests
adequately cover the code using `go test -cover`.

### Release procedure

1. Update version.go and internal/apmversion/version.go, and then run "make update-modules"
1. Update [`CHANGELOG.asciidoc`](changelog.asciidoc), by adding a new version heading (`==== 1.x.x - yyyy/MM/dd`) and changing the base tag of the Unreleased comparison URL
1. For major and minor releases, update the EOL table in [`upgrading.asciidoc`](docs/upgrading.asciidoc).
1. Merge changes into github.com/elastic/apm-agent-go@master
1. Create tags: vN.N.N, and module/$MODULE/vN.N.N for each instrumentation module

	scripts/tagversion.sh

1. Create release on GitHub

	hub release -d vN.N.N

1. Reset the latest major branch (1.x, 2.x etc) to point to the new release tag, e.g. git branch -f N.x vN.n.n
1. Update the latest major branch on upstream with `git push upstream <major_branch>`
