---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/getting-ready-new-protocol.html
---

# Getting Ready [getting-ready-new-protocol]

Packetbeat is written in [Go](http://golang.org/), so having Go installed and knowing the basics are prerequisites for understanding this guide. But don’t worry if you aren’t a Go expert. Go is a relatively new language, and very few people are experts in it. In fact, several people learned Go by contributing to Packetbeat and libbeat, including the original Packetbeat authors.

You will also need a good understanding of the wire protocol that you want to add support for. For standard protocols or protocols used in open source projects, you can usually find detailed specifications and example source code. Wireshark is a very useful tool for understanding the inner workings of the protocols it supports.

In some cases you can even make use of existing libraries for doing the actual parsing and decoding of the protocol. If the particular protocol has a Go implementation with a liberal enough license, you might be able to use it to parse and decode individual messages instead of writing your own parser.

Before starting, please also read the [*Contributing to Beats*](./index.md).


### Cloning and Compiling [_cloning_and_compiling]

After you have [installed Go](https://golang.org/doc/install) and set up the [GOPATH](https://golang.org/doc/code.md#GOPATH) environment variable to point to your preferred workspace location, you can clone Packetbeat with the following commands:

```shell
$ mkdir -p ${GOPATH}/src/github.com/elastic
$ cd ${GOPATH}/src/github.com/elastic
$ git clone https://github.com/elastic/beats.git
```

Note: If you have multiple go paths use `${GOPATH%%:*}`instead of `${GOPATH}`.

Then you can compile it with:

```shell
$ cd beats
$ make
```

Note that the location where you clone is important. If you prefer working outside of the `GOPATH` environment, you can clone to another directory and only create a symlink to the `$GOPATH/src/github.com/elastic/` directory.


## Forking and Branching [_forking_and_branching]

We recommend the following work flow for contributing to Packetbeat:

* Fork Beats in GitHub to your own account
* In the `$GOPATH/src/github.com/elastic/beats` folder, add your fork as a new remote. For example (replace `tsg` with your GitHub account):

```shell
$ git remote add tsg git@github.com:tsg/beats.git
```

* Create a new branch for your work:

```shell
$ git checkout -b cool_new_protocol
```

* Commit as often as you like, and then push to your private fork with:

```shell
$ git push --set-upstream tsg cool_new_protocol
```

* When you are ready to submit your PR, simply do so from the GitHub web interface. Feel free to submit your PR early. You can still add commits to the branch after creating the PR. Submitting the PR early gives us more time to provide feedback and perhaps help you with it.

