# elastic-agent-libs

This repository is the home to the common libraries used by Elastic Agent and Beats.

Provided packages:
* `github.com/elastic/elastic-agent-libs/api` Provides an HTTP API for debugging information.
* `github.com/elastic/elastic-agent-libs/api/npipe` Provides an API for debugging information via named pipes.
* `github.com/elastic/elastic-agent-libs/monitoring` Basic monitoring functionality used by Beats and Agent.
* `github.com/elastic/elastic-agent-libs/atomic` Atomic operations for integer and boolean types.
* `github.com/elastic/elastic-agent-libs/cloudid` is used for parsing `cloud.id` and `cloud.auth` when connecting to the Elastic stack.
* `github.com/elastic/elastic-agent-libs/config` the previous `config.go` file from `github.com/elastic/beats/v7/libbeat/common`. A minimal wrapper around `github.com/elastic/go-ucfg`. It contains helpers for merging and accessing configuration objects and flags.
* `github.com/elastic/elastic-agent-libs/file` is responsible for rotating and writing input and output files.
* `github.com/elastic/elastic-agent-libs/logp` is the well known logger from libbeat.
* `github.com/elastic/elastic-agent-libs/logp/cfgwarn` provides logging utilities for warning users about deprecated settings.
* `github.com/elastic/elastic-agent-libs/mapstr` is the old `github.com/elastic/beats/v7/libbeat/common.MapStr`. It is an extra layer on top of `map[string]interface{}`.
* `github.com/elastic/elastic-agent-libs/safemapstr` contains safe operations for `mapstr.M`.
* `github.com/elastic/elastic-agent-libs/str` the previous `stringset.go` file from `github.com/elastic/beats/v7/libbeat/common`. It provides a string set implementation. 
