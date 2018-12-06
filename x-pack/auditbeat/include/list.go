// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package include

import (
	// Include all Auditbeat modules so that they register their
	// factories with the global registry.
	_ "github.com/elastic/beats/x-pack/auditbeat/module/system/host"
	_ "github.com/elastic/beats/x-pack/auditbeat/module/system/packages"
	_ "github.com/elastic/beats/x-pack/auditbeat/module/system/process"
	_ "github.com/elastic/beats/x-pack/auditbeat/module/system/socket"
	_ "github.com/elastic/beats/x-pack/auditbeat/module/system/user"
)
