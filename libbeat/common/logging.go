package common

var maskList = MakeStringSet(
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

func applyLoggingMask(c interface{}) {
	switch cfg := c.(type) {
	case map[string]interface{}:
		for k, v := range cfg {
			if maskList.Has(k) {
				if arr, ok := v.([]interface{}); ok {
					for i := range arr {
						arr[i] = "xxxxx"
					}
				} else {
					cfg[k] = "xxxxx"
				}
			} else {
				applyLoggingMask(v)
			}
		}

	case []interface{}:
		for _, elem := range cfg {
			applyLoggingMask(elem)
		}
	}
}
