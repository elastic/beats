# Sarama fork for Beats

This repository holds the version of Sarama shipped with Beats, which sometimes includes bug fixes that have not yet reached an official Sarama release. The Beats package links against the `beats-fork` branch.

## Current state

The current baseline Sarama version is `v1.29.1`.

The additional patches applied to this version are:
- A fix for exponential backoff when a Kafka broker is down ([issue](https://github.com/Shopify/sarama/issues/1719), [pull request](https://github.com/elastic/sarama/pull/10), [upstream pull request](https://github.com/Shopify/sarama/pull/1720))

## Updating this repository

### Adding a new fix

If a new fix needs to be applied, open a pull request against the `beats-fork` branch of this repository. The fix should go through standard code review and should include an update to the "Current state" section of the `README.md`, linking to details of the change.

### Syncing with a more recent upstream

This fork does not always sync against the latest Sarama release. However, the intent is to stay as close to the current release as makes sense based on known issues and testing constraints.

Updates should always target an explicit release tag. To perform the update, first create your own fork of sarama with the "Fork" icon at the top right, then run the following from the command line:

    # Clone the repository locally, pointing at your own fork
    git clone git@github.com:[YOUR USERNAME]/sarama.git

    # Add links back to the Elastic and Shopify repositories so
    # we can merge between them, and run git fetch so we can
    # access the version tags.
    git remote add upstream git@github.com:elastic/sarama.git
    git remote add shopify git@github.com:Shopify/sarama.git
    git fetch --all

    # Check out the current Elastic fork and use it as a baseline
    # for a new local branch called "version-upgrade"
    git checkout upstream/beats-fork
    git checkout -b version-upgrade

    # Merge the target version into your new branch (replace
    # the version tag as appropriate).
    # If the new version conflicts with the outstanding bug
    # fixes, you will need to resolve those conflicts in this
    # step.
    git merge v1.26.4

    # Push the update back to your fork on github.
    git push --set-upstream origin version-upgrade

After this, open a regular pull request as in the previous section, remembering to update `README.md` to reflect the new baseline version.

### Updating the Beats repository

After a new fix or update, the Beats repository needs to be updated to point to the new version. You should almost always target the most recent commit in the [commits list for the `beats-fork` branch](https://github.com/elastic/sarama/commits/beats-fork) unless that would disrupt an impending release. Copy the commit hash (with the copy icon or by clicking through to the commit itself), then from a local branch of the Beats repository:

    go mod edit -replace github.com/Shopify/sarama=github.com/elastic/sarama@[commit hash]
    make notice

(When backporting to 7.8 or earlier you will also need to run `mage vendor` and then rerun `make notice` in the backport branch.)

You can then commit the results and submit a PR against the Beats repository, remembering to backport if appropriate. Pull requests MUST include the commit summary they target in their description, e.g.:

"This PR updates Sarama to the current Elastic fork, targeting the commit:

```
commit 123456789abcdefdf68d97cd255ebc039d36e88c (HEAD -> beats-fork, upstream/beats-fork)
Author: Someone <someone@somewhere.com>
Date:   Wed Jun 10 16:19:46 2020 -0400

    Apply some sort of fix
```
"

(This information is technically redundant, but makes it much easier for code reviewers to confirm that the right version is being applied.) See this [example PR](https://github.com/elastic/beats/pull/19527).

### What Sarama version is Beats linked to?

In any Beats branch, you can find the baseline Sarama version and any additional patches by checking the commit hash in the module configuration:

```
> grep Shopify/sarama go.mod

        github.com/Shopify/sarama v0.0.0-00010101000000-000000000000
        github.com/Shopify/sarama => github.com/elastic/sarama v1.19.1-0.20200625133446-b4d980d71f60
```

**Important**: the tag prefix on the Sarama fork, `"v1.19.1"` in this case, comes from internal interaction between git and go modules, and has no human-discernible connection to the deployed version!

To get the real answer, copy the hash at the end of the line (`b4d980d71f60` in this example), and check the repository status at that hash by opening `https://github.com/elastic/sarama/tree/[hash]`. The `README.md` at the bottom of that page will list the base version and patches that were in effect in its "Current state" section.

If there is no second line when you check `go.mod`, then the fork is not in use, and the current Sarama version is whatever is referenced by the tag at the end of the Sarama line.

### Turning the fork on / off in Beats

When a mainline Sarama release includes all fixes we need, we should generally prefer to link directly to it instead of using the fork. To do this:

- Submit a PR against the `beats-fork` branch, updating it to the new target version and reverting any remaining custom patches (see the instructions above). This is to make sure the fork is left in a clean state if it needs to be turned on again in the future.
- In the Beats repository:
  * select the new Sarama version with:

        go get github.com/Shopify/sarama@v1.2.3    [replace tag with the real target version]
  * Edit `go.mod` manually to remove the line `github.com/Shopify/sarama => github.com/elastic/sarama...`
  * Reload the linked version by running `mage vendor` and `mage update`
  * Commit the results and submit the resulting PR

Once this is done, the fork can be reenabled at any time by following instructions in the previous sections to update the fork and reference it from Beats.

Original `README.md` follows:

# sarama

[![Go Reference](https://pkg.go.dev/badge/github.com/Shopify/sarama.svg)](https://pkg.go.dev/github.com/Shopify/sarama)
[![Build Status](https://travis-ci.org/Shopify/sarama.svg?branch=master)](https://travis-ci.org/Shopify/sarama)
[![Coverage](https://codecov.io/gh/Shopify/sarama/branch/master/graph/badge.svg)](https://codecov.io/gh/Shopify/sarama)

Sarama is an MIT-licensed Go client library for [Apache Kafka](https://kafka.apache.org/) version 0.8 (and later).

## Getting started

- API documentation and examples are available via [pkg.go.dev](https://pkg.go.dev/github.com/Shopify/sarama).
- Mocks for testing are available in the [mocks](./mocks) subpackage.
- The [examples](./examples) directory contains more elaborate example applications.
- The [tools](./tools) directory contains command line tools that can be useful for testing, diagnostics, and instrumentation.

You might also want to look at the [Frequently Asked Questions](https://github.com/Shopify/sarama/wiki/Frequently-Asked-Questions).

## Compatibility and API stability

Sarama provides a "2 releases + 2 months" compatibility guarantee: we support
the two latest stable releases of Kafka and Go, and we provide a two month
grace period for older releases. This means we currently officially support
Go 1.15 through 1.16, and Kafka 2.6 through 2.8, although older releases are
still likely to work.

Sarama follows semantic versioning and provides API stability via the gopkg.in service.
You can import a version with a guaranteed stable API via http://gopkg.in/Shopify/sarama.v1.
A changelog is available [here](CHANGELOG.md).

## Contributing

- Get started by checking our [contribution guidelines](https://github.com/Shopify/sarama/blob/master/.github/CONTRIBUTING.md).
- Read the [Sarama wiki](https://github.com/Shopify/sarama/wiki) for more technical and design details.
- The [Kafka Protocol Specification](https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol) contains a wealth of useful information.
- For more general issues, there is [a google group](https://groups.google.com/forum/#!forum/kafka-clients) for Kafka client developers.
- If you have any questions, just ask!
