package common

var debugBlacklist = MakeStringSet(
	"password",
	"passphrase",
	"key_passphrase",
	"pass",
	"proxy_url",
	"url",
	"urls",
	"host",
	"hosts",
)

func filterDebugObject(c interface{}) {
	switch cfg := c.(type) {
	case map[string]interface{}:
		for k, v := range cfg {
			if debugBlacklist.Has(k) {
				if arr, ok := v.([]interface{}); ok {
					for i := range arr {
						arr[i] = "xxxxx"
					}
				} else {
					cfg[k] = "xxxxx"
				}
			} else {
				filterDebugObject(v)
			}
		}

	case []interface{}:
		for _, elem := range cfg {
			filterDebugObject(elem)
		}
	}
}
