// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoder

// Include supported protocols so that they can be registered
// into the protocol registry.

import (
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/ipfix"
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v1"
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v5"
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v6"
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v7"
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v8"
	_ "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)
