# elastic-agent-autodiscover

This repo contains packages required by autodiscover.

* `github.com/elastic/elastic-agent-autodiscover/bus`
* `github.com/elastic/elastic-agent-autodiscover/docker`
* `github.com/elastic/elastic-agent-autodiscover/kubernetes`
* `github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata`
* `github.com/elastic/elastic-agent-autodiscover/utils`


## Releasing updates

Note: For every user-facing change remember to update the [changelog](https://github.com/elastic/elastic-agent-autodiscover/blob/main/CHANGELOG.md) properly

Every time a new PR is merged and we want to make it available to external repos using this library we need to create a new tag.
Anybody with push privileges to this repository can create a new tag locally and push it to the upstream like the following:

```console
$ git remote -v
origin	git@github.com:ChrsMark/elastic-agent-autodiscover.git (fetch)
origin	git@github.com:ChrsMark/elastic-agent-autodiscover.git (push)
upstream	https://github.com/elastic/elastic-agent-autodiscover.git (fetch)
upstream	https://github.com/elastic/elastic-agent-autodiscover.git (push)
$ git tag -a v0.2.1 -m "New patch release for minor codebase improvements"
$ git push upstream v0.2.1 
Enumerating objects: 1, done.
Counting objects: 100% (1/1), done.
Writing objects: 100% (1/1), 190 bytes | 190.00 KiB/s, done.
Total 1 (delta 0), reused 0 (delta 0)
To https://github.com/elastic/elastic-agent-autodiscover.git
 * [new tag]             v0.2.1 -> v0.2.1
```

Then the tag should be available at https://github.com/elastic/elastic-agent-autodiscover/tags and anyone can use the the new version of the library in other projects. For example in order to use `v0.2.1` in Beats projects one would need a `go get github.com/elastic/elastic-agent-autodiscover@v0.2.1`.


After the tag is available a Release can be created using this tag and the proper content from the changelog.


## Development

When one wants to edit and test the library as part of the Beats or Elastic Agent projects, the local version of the dependency can be referenced with the following:

`go.mod`:
```golang
replace github.com/elastic/elastic-agent-autodiscover => /home/user/go/src/github.com/elastic/elastic-agent-autodiscover
```

This will use the local code rather than the upstream dependency. 
Note: Do not forget to exclude this change from the final commits.
