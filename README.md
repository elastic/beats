# elastic-agent-libs

This repository is the home to the common libraries used by Elastic Agent and Beats.

Provided packages:
* `github.com/elastic/elastic-agent-libs/config` the previous `config.go` file from `github.com/elastic/beats/v7/libbeat/common`. A minimal wrpper aroung `github.com/elastic/go-ucfg`. It contains helpers for merging and accessing configuration objects.
* `github.com/elastic/elastic-agent-libs/str` the previous `stringset.go` file from `github.com/elastic/beats/v7/libbeat/common`. It provides a string set implementation. 
