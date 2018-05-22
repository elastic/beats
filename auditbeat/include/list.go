package include

import (
	// Include all Auditbeat modules so that they register their
	// factories with the global registry.
	_ "github.com/elastic/beats/auditbeat/module/auditd"
	_ "github.com/elastic/beats/auditbeat/module/file_integrity"
)
