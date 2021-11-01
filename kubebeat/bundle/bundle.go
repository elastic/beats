package bundle

var Policies = map[string]string{
	"example.rego": `
				package authz

				default allow = false

				allow {
					input.open == "sesame"
				}
			`,
}

var Config = `{
		"services": {
			"test": {
				"url": %q
			}
		},
		"bundles": {
			"test": {
				"resource": "/bundles/bundle.tar.gz"
			}
		},
		"decision_logs": {
			"console": true
		}
	}`
