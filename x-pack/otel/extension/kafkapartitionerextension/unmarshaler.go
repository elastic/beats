// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// otelconsumer converts beat.Event.Fields (mapstr.M) into plog.Logs.
// Subsequently, kafka exporter uses raw unmarshaler to convert plog.Logs to []byte.
// We need to unmarshal it into mapstr.M again, as we need to hash the fields.
func unmarshalLogs(message []byte) (mapstr.M, error) {
	var val any
	if err := json.Unmarshal(message, &val); err != nil {
		return nil, err
	}

	switch v := val.(type) {
	case map[string]any:
		return mapstr.M(v), nil
	case mapstr.M:
		return v, nil
	}
	return nil, fmt.Errorf("unmarshaled value shoud be a map, found %T", val)
}
