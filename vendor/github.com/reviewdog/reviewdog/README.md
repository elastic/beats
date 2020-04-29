<p align="center">
  <a href="https://github.com/reviewdog/reviewdog">
    <img alt="reviewdog" src="https://raw.githubusercontent.com/haya14busa/i/d598ed7dc49fefb0018e422e4c43e5ab8f207a6b/reviewdog/reviewdog.logo.png">
  </a>
</p>

<h2 align="center">
  reviewdog - A code review dog who keeps your codebase healthy.
</h2>

<p align="center">
  <a href="https://github.com/reviewdog/reviewdog/blob/master/LICENSE">
    <img alt="LICENSE" src="https://img.shields.io/badge/license-MIT-blue.svg?maxAge=43200">
  </a>
  <a href="https://godoc.org/github.com/reviewdog/reviewdog">
    <img alt="GoDoc" src="https://img.shields.io/badge/godoc-reference-4F73B3.svg?label=godoc.org&maxAge=43200">
  </a>
  <a href="https://github.com/reviewdog/reviewdog/releases">
    <img alt="releases" src="https://img.shields.io/github/release/reviewdog/reviewdog.svg?maxAge=43200">
  </a>
  <a href="https://github.com/reviewdog/reviewdog/releases">
    <img alt="Github All Releases" src="https://img.shields.io/github/downloads/reviewdog/reviewdog/total.svg?maxAge=43200">
  </a>
</p>

<p align="center">
  <a href="https://github.com/reviewdog/reviewdog/actions?query=workflow%3AGo+event%3Apush+branch%3Amaster">
    <img alt="GitHub Actions" src="https://github.com/reviewdog/reviewdog/workflows/Go/badge.svg">
  </a>
  <a href="https://github.com/reviewdog/reviewdog/actions?query=workflow%3Areviewdog+event%3Apush+branch%3Amaster">
    <img alt="reviewdog" src="https://github.com/reviewdog/reviewdog/workflows/reviewdog/badge.svg?branch=master&event=push">
  </a>
  <a href="https://travis-ci.org/reviewdog/reviewdog"><img alt="Travis Status" src="https://img.shields.io/travis/reviewdog/reviewdog/master.svg?label=travis&maxAge=43200"></a>
  <a href="https://circleci.com/gh/reviewdog/reviewdog"><img alt="CircleCI Status" src="https://img.shields.io/circleci/project/github/reviewdog/reviewdog/master.svg?label=circle&maxAge=43200"></a>
  <a href="https://codecov.io/github/reviewdog/reviewdog"><img alt="Coverage Status" src="https://img.shields.io/codecov/c/github/reviewdog/reviewdog/master.svg?maxAge=43200"></a>
  <a href="https://starcharts.herokuapp.com/reviewdog/reviewdog"><img alt="Stars" src="https://img.shields.io/github/stars/reviewdog/reviewdog.svg?style=social&maxAge=43200"></a>
</p>

"reviewdog" provides a way to post review comments to code hosting service,
such as GitHub, automatically by integrating with any linter tools with ease.
It uses an output of lint tools and posts them as a comment if findings are in
diff of patches to review.

reviewdog also supports run in the local environment to filter an output of lint tools
by diff.

[design doc](https://docs.google.com/document/d/1mGOX19SSqRowWGbXieBfGPtLnM0BdTkIc9JelTiu6wA/edit?usp=sharing)

## Table of Contents

- [Installation](#installation)
- [Input Format](#input-format)
  * ['errorformat'](#errorformat)
  * [Available pre-defined 'errorformat'](#available-pre-defined-errorformat)
  * [checkstyle format](#checkstyle-format)
- [reviewdog config file](#reviewdog-config-file)
- [Reporters](#reporters)
  * [Reporter: Local (-reporter=local) [default]](#reporter-local--reporterlocal-default)
  * [Reporter: GitHub Checks (-reporter=github-pr-check)](#reporter-github-checks--reportergithub-pr-check)
  * [Reporter: GitHub Checks (-reporter=github-check)](#reporter-github-checks--reportergithub-check)
  * [Reporter: GitHub PullRequest review comment (-reporter=github-pr-review)](#reporter-github-pullrequest-review-comment--reportergithub-pr-review)
  * [Reporter: GitLab MergeRequest discussions (-reporter=gitlab-mr-discussion)](#reporter-gitlab-mergerequest-discussions--reportergitlab-mr-discussion)
  * [Reporter: GitLab MergeRequest commit (-reporter=gitlab-mr-commit)](#reporter-gitlab-mergerequest-commit--reportergitlab-mr-commit)
- [Supported CI services](#supported-ci-services)
  * [GitHub Actions](#github-actions)
  * [Travis CI](#travis-ci)
  * [Circle CI](#circle-ci)
  * [GitLab CI](#gitlab-ci)
  * [Common (Jenkins, local, etc...)](#common-jenkins-local-etc)
    + [Jenkins with Github pull request builder plugin](#jenkins-with-github-pull-request-builder-plugin)
- [Articles](#articles)

[![github-pr-check sample](https://user-images.githubusercontent.com/3797062/40884858-6efd82a0-6756-11e8-9f1a-c6af4f920fb0.png)](https://github.com/reviewdog/reviewdog/pull/131/checks)
![comment in pull-request](https://user-images.githubusercontent.com/3797062/40941822-1d775064-6887-11e8-98e9-4775d37d47f8.png)
![commit status](https://user-images.githubusercontent.com/3797062/40941738-d62acb0a-6886-11e8-858d-7b97aded2a42.png)
[![sample-comment.png](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/sample-comment.png)](https://github.com/reviewdog/reviewdog/pull/24#discussion_r84599728)
![reviewdog-local-demo.gif](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/reviewdog-local-demo.gif)

## Installation

```shell
# Install latest version. (Install it into ./bin/ by default).
$ curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s

# Specify installation directory ($(go env GOPATH)/bin/) and version.
$ curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- -b $(go env GOPATH)/bin [vX.Y.Z]

# In alpine linux (as it does not come with curl by default)
$ wget -O - -q https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s [vX.Y.Z]
```

or

```shell
$ go get -u github.com/reviewdog/reviewdog/cmd/reviewdog
```

### homebrew / linuxbrew
You can also install reviewdog using brew:

```shell
$ brew install reviewdog/tap/reviewdog
$ brew upgrade reviewdog/tap/reviewdog
```

## Input Format

### 'errorformat'

reviewdog accepts any compiler or linter result from stdin and parses it with
scan-f like [**'errorformat'**](https://github.com/reviewdog/errorformat),
which is the port of Vim's [errorformat](https://vim-jp.org/vimdoc-en/quickfix.html#error-file-format)
feature.

For example, if the result format is `{file}:{line number}:{column number}: {message}`,
errorformat should be `%f:%l:%c: %m` and you can pass it as `-efm` arguments.

```shell
$ golint ./...
comment_iowriter.go:11:6: exported type CommentWriter should have comment or be unexported
$ golint ./... | reviewdog -efm="%f:%l:%c: %m" -diff="git diff master"
```

| name | description |
| ---- | ----------- |
| %f | file name |
| %l | line number |
| %c | column number |
| %m | error message |
| %% | the single '%' character |
| ... | ... |

Please see [reviewdog/errorformat](https://github.com/reviewdog/errorformat)
and [:h errorformat](https://vim-jp.org/vimdoc-en/quickfix.html#error-file-format)
if you want to deal with a more complex output. 'errorformat' can handle more
complex output like a multi-line error message.

You can also try errorformat on [the Playground](https://reviewdog.github.io/errorformat-playground/)!

By this 'errorformat' feature, reviewdog can support any tools output with ease.

### Available pre-defined 'errorformat'

But, you don't have to write 'errorformat' in many cases. reviewdog supports
pre-defined errorformat for major tools.

You can find available errorformat name by `reviewdog -list` and you can use it
with `-f={name}`.

```shell
$ reviewdog -list
golint          linter for Go source code                                       - https://github.com/golang/lint
govet           Vet examines Go source code and reports suspicious problems     - https://golang.org/cmd/vet/
sbt             the interactive build tool                                      - http://www.scala-sbt.org/
...
```

```shell
$ golint ./... | reviewdog -f=golint -diff="git diff master"
```

You can add supported pre-defined 'errorformat' by contributing to [reviewdog/errorformat](https://github.com/reviewdog/errorformat)

### checkstyle format

reviewdog also accepts [checkstyle XML format](http://checkstyle.sourceforge.net/) as well.
If the linter supports checkstyle format as a report format, you can use
-f=checkstyle instead of using 'errorformat'.

```shell
# Local
$ eslint -f checkstyle . | reviewdog -f=checkstyle -diff="git diff"

# CI (overwrite tool name which is shown in review comment by -name arg)
$ eslint -f checkstyle . | reviewdog -f=checkstyle -name="eslint" -reporter=github-pr-check
```

Also, if you want to pass other Json/XML/etc... format to reviewdog, you can write a converter.

```shell
$ <linter> | <convert-to-checkstyle> | reviewdog -f=checkstyle -name="<linter>" -reporter=github-pr-check
```

## reviewdog config file

reviewdog can also be controlled via the .reviewdog.yml configuration file instead of "-f" or "-efm" arguments.

With .reviewdog.yml, you can run the same commands both CI service and local
environment including editor integration with ease.

#### .reviewdog.yml

```yaml
runner:
  <tool-name>:
    cmd: <command> # (required)
    errorformat: # (optional if there is supported format for <tool-name>. see reviewdog -list)
      - <list of errorformat>
    name: <tool-name> # (optional. you can overwrite <tool-name> defined by runner key)
    level: <level> # (optional. same as -level flag. [info,warning,error])

  # examples
  golint:
    cmd: golint ./...
    errorformat:
      - "%f:%l:%c: %m"
    level: warning
  govet:
    cmd: go tool vet -all -shadowstrict .
```

```shell
$ reviewdog -diff="git diff master"
project/run_test.go:61:28: [golint] error strings should not end with punctuation
project/run.go:57:18: [errcheck]        defer os.Setenv(name, os.Getenv(name))
project/run.go:58:12: [errcheck]        os.Setenv(name, "")
# You can use -runners to run only specified runners.
$ reviewdog -diff="git diff master" -runners=golint,govet
project/run_test.go:61:28: [golint] error strings should not end with punctuation
# You can use -conf to specify config file path.
$ reviewdog -conf=./.reviewdog.yml -reporter=github-pr-check
```

Output format for project config based run is one of the following formats.

- `<file>: [<tool name>] <message>`
- `<file>:<lnum>: [<tool name>] <message>`
- `<file>:<lnum>:<col>: [<tool name>] <message>`

## Reporters

reviewdog can report results both in local environment and review services as
continuous integration.

### Reporter: Local (-reporter=local) [default]

reviewdog can find newly introduced findings by filtering linter results
using diff. You can pass diff command as `-diff` arg.

```shell
$ golint ./... | reviewdog -f=golint -diff="git diff master"
```

### Reporter: GitHub Checks (-reporter=github-pr-check)

[![github-pr-check sample annotation with option 1](https://user-images.githubusercontent.com/3797062/64875597-65016f80-d688-11e9-843f-4679fb666f0d.png)](https://github.com/reviewdog/reviewdog/pull/275/files#annotation_6177941961779419)
[![github-pr-check sample](https://user-images.githubusercontent.com/3797062/40884858-6efd82a0-6756-11e8-9f1a-c6af4f920fb0.png)](https://github.com/reviewdog/reviewdog/pull/131/checks)

github-pr-check reporter reports results to [GitHub Checks](https://help.github.com/articles/about-status-checks/).

You can change report level for this reporter by `level` field in [config
file](#reviewdog-config-file) or `-level` flag. You can control GitHub status
check result with this feature. (default: error)

| Level     | GitHub Status |
| --------- | ------------- |
| `info`    | neutral       |
| `warning` | neutral       |
| `error`   | failure       |

There are two options to use this reporter.

#### Option 1) Run reviewdog from GitHub Actions w/ secrets.GITHUB_TOKEN

Example: [.github/workflows/reviewdog.yml](.github/workflows/reviewdog.yml)

```yaml
- name: Run reviewdog
  env:
    REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    golint ./... | reviewdog -f=golint -reporter=github-pr-check
```

See [GitHub Actions](#github-actions) section too. You can also use public
reviewdog GitHub Actions.

#### Option 2) Install reviewdog GitHub Apps
reviewdog CLI send a request to reviewdog GitHub App server and the server post
results as GitHub Checks, because Check API only supported for GitHub App and
GitHub Actions.

1. Install reviewdog Apps. https://github.com/apps/reviewdog
2. Set `REVIEWDOG_TOKEN` or run reviewdog CLI in trusted CI providers.
  - Get token from `https://reviewdog.app/gh/{owner}/{repo-name}`.

```shell
$ export REVIEWDOG_TOKEN="<token>"
$ reviewdog -reporter=github-pr-check
```

Note: Token is not required if you run reviewdog in Travis or AppVeyor.

*Caution*

As described above, github-pr-check reporter with Option 2 is depending on
reviewdog GitHub App server.
The server is running with haya14busa's pocket money for now and I may break
things, so I cannot ensure that the server is running 24h and 365 days.

**UPDATE:** Started getting support by [opencollective](https://opencollective.com/reviewdog)
and GitHub sponsor.
See [Supporting reviewdog](#supporting-reviewdog)

github-pr-check reporter is better than github-pr-review reporter in general
because it provides more rich feature and has less scope, but please bear in
mind the above caution and please use it on your own risk.

You can use github-pr-review reporter if you don't want to depend on reviewdog
server.

### Reporter: GitHub Checks (-reporter=github-check)

It's basically same as `-reporter=github-pr-check` except it works not only for
Pull Request but also for commit and it reports results outside Pull Request
diff too.

[![sample comment outside diff](https://user-images.githubusercontent.com/3797062/69917921-e0680580-14ae-11ea-9a56-de9e3cbac005.png)](https://github.com/reviewdog/reviewdog/pull/364/files)

You can create [reviewdog badge](#reviewdog-badge-) for this reporter.

### Reporter: GitHub PullRequest review comment (-reporter=github-pr-review)

[![sample-comment.png](https://raw.githubusercontent.com/haya14busa/i/dc0ccb1e110515ea407c146d99b749018db05c45/reviewdog/sample-comment.png)](https://github.com/reviewdog/reviewdog/pull/24#discussion_r84599728)

github-pr-review reporter reports results to GitHub PullRequest review comments
using GitHub Personal API Access Token.
[GitHub Enterprise](https://enterprise.github.com/home) is supported too.

- Go to https://github.com/settings/tokens and generate new API token.
- Check `repo` for private repositories or `public_repo` for public repositories.

```shell
$ export REVIEWDOG_GITHUB_API_TOKEN="<token>"
$ reviewdog -reporter=github-pr-review
```

For GitHub Enterprise, set API endpoint by environment variable.

```shell
$ export GITHUB_API="https://example.githubenterprise.com/api/v3/"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need to skip verifying SSL
```

See [GitHub Actions](#github-actions) section too if you can use GitHub
Actions. You can also use public reviewdog GitHub Actions.

### Reporter: GitLab MergeRequest discussions (-reporter=gitlab-mr-discussion)

[![gitlab-mr-discussion sample](https://user-images.githubusercontent.com/3797062/41810718-f91bc540-773d-11e8-8598-fbc09ce9b1c7.png)](https://gitlab.com/haya14busa/reviewdog/merge_requests/113#note_83411103)

Required GitLab version: >= v10.8.0

gitlab-mr-discussion reporter reports results to GitLab MergeRequest discussions using
GitLab Personal API Access token.
Get the token with `api` scope from https://gitlab.com/profile/personal_access_tokens.

```shell
$ export REVIEWDOG_GITLAB_API_TOKEN="<token>"
$ reviewdog -reporter=gitlab-mr-discussion
```

For self-hosted GitLab, set API endpoint by environment variable.

```shell
$ export GITLAB_API="https://example.gitlab.com/api/v4"
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need to skip verifying SSL
```

### Reporter: GitLab MergeRequest commit (-reporter=gitlab-mr-commit)

gitlab-mr-commit is similar to [gitlab-mr-discussion](#reporter-gitlab-mergerequest-discussions--reportergitlab-mr-discussion) reporter but reports results to each commit in GitLab MergeRequest.

gitlab-mr-discussion is recommended, but you can use gitlab-mr-commit reporter
if your GitLab version is under v10.8.0.

```shell
$ export REVIEWDOG_GITLAB_API_TOKEN="<token>"
$ reviewdog -reporter=gitlab-mr-commit
```

## Supported CI services

### [GitHub Actions](https://github.com/features/actions)

Example: [.github/workflows/reviewdog.yml](.github/workflows/reviewdog.yml)

```yaml
name: reviewdog
on: [pull_request]
jobs:
  reviewdog:
    name: reviewdog
    runs-on: ubuntu-latest
    steps:
      # ...
      - name: Setup reviewdog
        run: |
          mkdir -p $HOME/bin && curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- -b $HOME/bin
          echo ::add-path::$HOME/bin
          echo ::add-path::$(go env GOPATH)/bin # for Go projects
      - name: Run reviewdog
        env:
          REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          reviewdog -reporter=github-pr-check -runners=golint,govet
          # or
          reviewdog -reporter=github-pr-review -runners=golint,govet
```

<details>
<summary><strong>Example (github-check reporter):</strong></summary>

[.github/workflows/reviewdog](.github/workflows/reviewdog.yml)

Only `github-check` reporter can run on push event too.

```yaml
name: reviewdog (github-check)
on:
  push:
    branches:
      - master
  pull_request:

jobs:
  reviewdog:
    name: reviewdog
    runs-on: ubuntu-latest
    steps:
      # ...
      - name: Run reviewdog
        env:
          REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          reviewdog -reporter=github-check -runners=golint,govet
```

</details>

#### Public Reviewdog GitHub Actions
You can also use public GitHub Actions to start using reviewdog with ease! :tada: :arrow_forward: :tada:

[Marketplace](https://github.com/marketplace?utf8=✓&type=actions&query=reviewdog)

- Common
  - [reviewdog/action-misspell](https://github.com/reviewdog/action-misspell) - Run [misspell](https://github.com/client9/misspell).
  - [tsuyoshicho/action-textlint](https://github.com/tsuyoshicho/action-textlint) - Run [textlint](https://github.com/textlint/textlint)
  - [tsuyoshicho/action-redpen](https://github.com/tsuyoshicho/action-redpen) - Run [redpen](https://github.com/redpen-cc/redpen)
- Docker
  - [reviewdog/action-hadolint](https://github.com/reviewdog/action-hadolint) - Run [hadolint](https://github.com/hadolint/hadolint) to lint `Dockerfile`.
- Env
  - [mgrachev/action-dotenv-linter](https://github.com/mgrachev/action-dotenv-linter) - Run [dotenv-linter](https://github.com/mgrachev/dotenv-linter) to lint `.env` files.
- Shell script
  - [reviewdog/action-shellcheck](https://github.com/reviewdog/action-shellcheck) - Run [shellcheck](https://github.com/koalaman/shellcheck).
- Go
  - [reviewdog/action-golangci-lint](https://github.com/reviewdog/action-golangci-lint) - Run [golangci-lint](https://github.com/golangci/golangci-lint) and supported linters individually by golangci-lint.
- JavaScript
  - [reviewdog/action-eslint](https://github.com/reviewdog/action-eslint) - Run [eslint](https://github.com/eslint/eslint).
- CSS
  - [reviewdog/action-stylelint](https://github.com/reviewdog/action-stylelint) - Run [stylelint](https://github.com/stylelint/stylelint).
- Vim script
  - [reviewdog/action-vint](https://github.com/reviewdog/action-vint) - Run [vint](https://github.com/Kuniwak/vint).
  - [tsuyoshicho/action-vimlint](https://github.com/tsuyoshicho/action-vimlint) - Run [vim-vimlint](https://github.com/syngan/vim-vimlint)
- Terraform
  - [reviewdog/action-tflint](https://github.com/reviewdog/action-tflint) - Run [tflint](https://github.com/wata727/tflint).
- YAML
  - [reviewdog/action-yamllint](https://github.com/reviewdog/action-yamllint) - Run [yamllint](https://github.com/adrienverge/yamllint).
- Ruby
  - [reviewdog/action-rubocop](https://github.com/reviewdog/action-rubocop) - Run [rubocop](https://github.com/rubocop-hq/rubocop).
- Python
  - [wemake-python-styleguide](https://github.com/wemake-services/wemake-python-styleguide) - Run wemake-python-styleguide

Please open a Pull Request to add your created reviewdog actions here :sparkles:.
I can also consider to put your created repositories under reviewdog org and co-maintain the actions.
Example: [action-tflint](https://github.com/reviewdog/reviewdog/issues/322).

#### Graceful Degradation for Pull Requests from forked repositories

![Graceful Degradation example](https://user-images.githubusercontent.com/3797062/71781334-e2266b00-3010-11ea-8a38-dee6e30c8162.png)

`GITHUB_TOKEN` for Pull Requests from forked repository doesn't have write
access to Check API nor Review API due to [GitHub Actions
restriction](https://help.github.com/en/articles/virtual-environments-for-github-actions#github_token-secret).

Instead, reviewdog uses [Logging commands of GitHub
Actions](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/development-tools-for-github-actions#set-an-error-message-error)
to post results as
[annotations](https://developer.github.com/v3/checks/runs/#annotations-object)
similar to `github-pr-check` reporter.

Note that there is a limitation for annotations created by logging commands,
such as [max # of annotations per run](https://github.com/reviewdog/reviewdog/issues/411#issuecomment-570893427).
You can check GitHub Actions log to see full results in such cases.

#### reviewdog badge [![reviewdog](https://github.com/reviewdog/reviewdog/workflows/reviewdog/badge.svg?branch=master&event=push)](https://github.com/reviewdog/reviewdog/actions?query=workflow%3Areviewdog+event%3Apush+branch%3Amaster)

As [`github-check` reporter](#reporter-github-checks--reportergithub-pr-check) support running on commit, we can create reviewdog
[GitHub Action badge](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/configuring-a-workflow#adding-a-workflow-status-badge-to-your-repository)
to check the result against master commit for example. :tada:

Example:
```
<!-- Replace <OWNWER> and <REPOSITORY>. It assumes workflow name is "reviewdog" -->
[![reviewdog](https://github.com/<OWNER>/<REPOSITORY>/workflows/reviewdog/badge.svg?branch=master&event=push)](https://github.com/<OWNER>/<REPOSITORY>/actions?query=workflow%3Areviewdog+event%3Apush+branch%3Amaster)
```

### Travis CI

#### Travis CI (-reporter=github-pr-check)

If you use -reporter=github-pr-check in Travis CI, you don't need to set `REVIEWDOG_TOKEN`.

Example:

```yaml
install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- -b ~/bin

script:
  - reviewdog -conf=.reviewdog.yml -reporter=github-pr-check
```

#### Travis CI (-reporter=github-pr-review)

Store GitHub API token by [travis encryption keys](https://docs.travis-ci.com/user/encryption-keys/).

```shell
$ gem install travis
$ travis encrypt REVIEWDOG_GITHUB_API_TOKEN=<token> --add env.global
```
Example:

```yaml
env:
  global:
    - secure: <token>

install:
  - mkdir -p ~/bin/ && export export PATH="~/bin/:$PATH"
  - curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- -b ~/bin

script:
  - >-
    golint ./... | reviewdog -f=golint -reporter=github-pr-review
```

Examples
- https://github.com/azu/textlint-reviewdog-example

### Circle CI

Store `REVIEWDOG_TOKEN` or `REVIEWDOG_GITHUB_API_TOKEN` in
[Environment variables - CircleCI](https://circleci.com/docs/environment-variables/#setting-environment-variables-for-all-commands-without-adding-them-to-git)

#### .circleci/config.yml sample

```yaml
version: 2
jobs:
  build:
    docker:
      - image: golang:latest
    steps:
      - checkout
      - run: curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- -b ./bin
      - run: go vet ./... 2>&1 | ./bin/reviewdog -f=govet -reporter=github-pr-check
      # or
      - run: go vet ./... 2>&1 | ./bin/reviewdog -f=govet -reporter=github-pr-review
```

### GitLab CI

Store `REVIEWDOG_GITLAB_API_TOKEN` in [GitLab CI variable](https://docs.gitlab.com/ee/ci/variables/#variables).

#### .gitlab-ci.yml sample

```yaml
reviewdog:
  script:
    - reviewdog -reporter=gitlab-mr-discussion
    # Or
    - reviewdog -reporter=gitlab-mr-commit
```

### Common (Jenkins, local, etc...)

You can use reviewdog to post review comments from anywhere with following
environment variables.

| name | description |
| ---- | ----------- |
| `CI_PULL_REQUEST` | Pull Request number (e.g. 14) |
| `CI_COMMIT`       | SHA1 for the current build |
| `CI_REPO_OWNER`   | repository owner (e.g. "haya14busa" for https://github.com/haya14busa/reviewdog) |
| `CI_REPO_NAME`    | repository name (e.g. "reviewdog" for https://github.com/haya14busa/reviewdog) |
| `CI_BRANCH`       | [optional] branch of the commit |

```shell
$ export CI_PULL_REQUEST=14
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=$(git rev-parse HEAD)
```
and set a token if required.

```shell
$ REVIEWDOG_TOKEN="<token>"
$ REVIEWDOG_GITHUB_API_TOKEN="<token>"
$ REVIEWDOG_GITLAB_API_TOKEN="<token>"
```

If a CI service doesn't provide information such as Pull Request ID - reviewdog can guess it by branch name and commit SHA.
Just pass the flag `guess`:

```shell
$ reviewdog -conf=.reviewdog.yml -reporter=github-pr-check -guess
```

#### Jenkins with Github pull request builder plugin
- [GitHub pull request builder plugin - Jenkins - Jenkins Wiki](https://wiki.jenkins-ci.org/display/JENKINS/GitHub+pull+request+builder+plugin)

```shell
$ export CI_PULL_REQUEST=${ghprbPullId}
$ export CI_REPO_OWNER=haya14busa
$ export CI_REPO_NAME=reviewdog
$ export CI_COMMIT=${ghprbActualCommit}
$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true # set this as you need
$ REVIEWDOG_TOKEN="<token>" reviewdog -reporter=github-pr-check
# Or
$ REVIEWDOG_GITHUB_API_TOKEN="<token>" reviewdog -reporter=github-pr-review
```

## Articles
- [reviewdog — A code review dog who keeps your codebase healthy ](https://medium.com/@haya14busa/reviewdog-a-code-review-dog-who-keeps-your-codebase-healthy-d957c471938b)
- [reviewdog ♡ GitHub Check — improved automated review experience](https://medium.com/@haya14busa/reviewdog-github-check-improved-automated-review-experience-58f89e0c95f3)
- [Automated Code Review on GitHub Actions with reviewdog for any languages/tools](https://medium.com/@haya14busa/automated-code-review-on-github-actions-with-reviewdog-for-any-languages-tools-20285e04448e)

## :bird: Author
haya14busa [![GitHub followers](https://img.shields.io/github/followers/haya14busa.svg?style=social&label=Follow)](https://github.com/haya14busa)

## Contributors

[![Contributors](https://opencollective.com/reviewdog/contributors.svg?width=890)](https://github.com/reviewdog/reviewdog/graphs/contributors)

### Supporting reviewdog

Become GitHub Sponsor for [each contributor](https://github.com/reviewdog/reviewdog/graphs/contributors)
or become a backer or sponsor from [opencollective](https://opencollective.com/reviewdog).

[![Become a backer](https://opencollective.com/reviewdog/tiers/backer.svg?avatarHeight=64)](https://opencollective.com/reviewdog#backers)
