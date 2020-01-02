# Contributing to GoFormation

✨ Thanks for contributing to **GoFormation**! ✨

As a contributor, here are the guidelines we would like you to follow:
- [Code of conduct](#code-of-conduct)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Coding rules](#coding-rules)
  - [Source code](#source-code)
  - [Commit message guidelines](#commit-message-guidelines)
    - [Atomic commits](#atomic-commits)
    - [Commit message format](#commit-message-format)
    - [Revert](#revert)
    - [Type](#type)
    - [Subject](#subject)
    - [Body](#body)
    - [Footer](#footer)
    - [Examples](#examples)
- [Working with the code](#working-with-the-code)
  - [Set up the workspace](#set-up-the-workspace)
  - [Tests](#tests)
  - [Commits](#commits)
  - [Generating AWS CloudFormation Resources](#generating-aws-cloudformation-resources)

We also recommend that you read [How to Contribute to Open Source](https://opensource.guide/how-to-contribute).

## Code of conduct

Help us keep **GoFormation** open and inclusive. Please read and follow our [Code of conduct](CODE_OF_CONDUCT.md).

## Submitting a Pull Request

Good pull requests, whether patches, improvements, or new features, are a fantastic help. They should remain focused in scope and avoid containing unrelated commits.

**Please ask first** before embarking on any significant pull requests (e.g. implementing features, refactoring code), otherwise you risk spending a lot of time working on something that the project's developers might not want to merge into the project.

If you have never created a pull request before, welcome 🎉 😄. [Here is a great tutorial](https://opensource.guide/how-to-contribute/#opening-a-pull-request) on how to send one :)

Here is a summary of the steps to follow:

1. [Set up the workspace](#set-up-the-workspace)
2. If you cloned a while ago, get the latest changes from upstream and update dependencies:
```bash
$ git checkout master
$ git pull upstream master
```
3. Create a new topic branch (off the main project development branch) to contain your feature, change, or fix:
```bash
$ git checkout -b <topic-branch-name>
```
4. Make your code changes, following the [Coding rules](#coding-rules)
5. Push your topic branch up to your fork:
```bash
$ git push origin <topic-branch-name>
```
6. [Open a Pull Request](https://help.github.com/articles/creating-a-pull-request/#creating-the-pull-request) with a clear title and description.

**Tips**:
- For ambitious tasks, open a Pull Request as soon as possible with the `[WIP]` prefix in the title, in order to get feedback and help from the community.
- [Allow GoFormation maintainers to make changes to your Pull Request branch](https://help.github.com/articles/allowing-changes-to-a-pull-request-branch-created-from-a-fork). This way, we can rebase it and make some minor changes if necessary. 

## Coding rules

### Source code

To ensure consistency and quality throughout the source code, all code modifications must have:
- A [test](#tests) for every possible case introduced by your code change
- [Valid commit message(s)](#commit-message-guidelines)
- Documentation for new features
- Updated documentation for modified features

### Commit message guidelines

#### Atomic commits

If possible, make [atomic commits](https://en.wikipedia.org/wiki/Atomic_commit), which means:
- a commit should contain exactly one self-contained functional change
- a functional change should be contained in exactly one commit
- a commit should not create an inconsistent state (such as test errors, linting errors, partial fix, feature with documentation etc...)

A complex feature can be broken down into multiple commits as long as each one maintains a consistent state and consists of a self-contained change.

#### Commit message format

Each commit message consists of a **header**, a **body** and a **footer**. The header has a special format that includes a **type**, a **scope** and a **subject**:

```commit
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

The **header** is mandatory and the **scope** of the header is optional.

The **footer** can contain a [closing reference to an issue](https://help.github.com/articles/closing-issues-via-commit-messages).

#### Revert

If the commit reverts a previous commit, it should begin with `revert: `, followed by the header of the reverted commit. In the body it should say: `This reverts commit <hash>.`, where the hash is the SHA of the commit being reverted.

#### Type

The type must be one of the following:

| Type         | Description                                                                                                 |
| ------------ | ----------------------------------------------------------------------------------------------------------- |
| **build**    | Changes that affect the build system or external dependencies (go)                                          |
| **ci**       | Changes to our CI configuration files and scripts (example scopes: Travis, Circle, BrowserStack, SauceLabs) |
| **docs**     | Documentation only changes                                                                                  |
| **feat**     | A new feature                                                                                               |
| **fix**      | A bug fix                                                                                                   |
| **perf**     | A code change that improves performance                                                                     |
| **refactor** | A code change that neither fixes a bug nor adds a feature                                                   |
| **style**    | Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)      |
| **test**     | Adding missing tests or correcting existing tests                                                           |

#### Subject

The subject contains succinct description of the change:

- use the imperative, present tense: "change" not "changed" nor "changes"
- don't capitalize first letter
- no dot (.) at the end

#### Body
Just as in the **subject**, use the imperative, present tense: "change" not "changed" nor "changes".
The body should include the motivation for the change and contrast this with previous behavior.

#### Footer
The footer should contain any information about **Breaking Changes** and is also the place to reference GitHub issues that this commit **Closes**.

**Breaking Changes** should start with the word `BREAKING CHANGE:` with a space or two newlines. The rest of the commit message is then used for this.

#### Examples

```commit
`fix(pencil): stop graphite breaking when too much pressure applied`
```

```commit
`feat(pencil): add 'graphiteWidth' option`

Fix #42
```

```commit
perf(pencil): remove graphiteWidth option`

BREAKING CHANGE: The graphiteWidth option has been removed.

The default graphite width of 10mm is always used for performance reasons.
```

## Working with the code

### Set up the workspace

[Fork](https://guides.github.com/activities/forking/#fork) the project, [clone](https://guides.github.com/activities/forking/#clone) your fork, configure the remotes and install the dependencies:

```bash
# Clone your fork of the repo into the current directory
$ git clone https://github.com/<your-github-user>/goformation
# Navigate to the newly cloned directory
$ cd goformation
# Assign the original repo to a remote called "upstream"
$ git remote add upstream https://github.com/awslabs/goformation
```

### Tests

Before pushing your code changes make sure all **tests pass**.

```bash
$ go test ./...
```

### Commits

The [GoFormation](https://github.com/awslabs/goformation) repository uses [semantic-release](https://github.com/semantic-release/semantic-release) to automatically generate CHANGELOG entries, and cut releases based on commit messages. It's important to follow the [commit message guidelines](#commit-message-guidelines) so that this process continues to work.

To make things easier, you can use a tool like [Commitizen CLI](https://github.com/commitizen/cz-cli) to help you craft your commit messages. 

After staging your changes with `git add`, run `npx git-cz` to start the interactive commit message CLI.

### Generating AWS CloudFormation Resources

While you work on features, you may want to regenerate the CloudFormation resources in GoFormation. To do this, run:

```
$ go generate
GoFormation Resource Generator
Downloading cloudformation specification from https://d1uauaxba7bl26.cloudfront.net/latest/gzip/CloudFormationResourceSpecification.json
Downloading sam specification from file://generate/sam-2016-10-31.json
Updated the following AWS CloudFormation resources:
 - AWS::Serverless::Application
 - AWS::SNS::Topic
Processed 1161 resources

```
You will see a summary of the resources that have been updated, and for more detailed changes you can just run `git diff`.

If your contributions to GoFormation include regenerating resources (e.g. you make a change to the resource template, which modifies all resources), please make sure to run the `go generate` in a different git commit. This will make the pull request review process a lot easier for everybody involved :)