package bundle

import (
	"embed"
	"io/fs"
	"os"
	"strings"

	"github.com/elastic/beats/v7/libbeat/logp"
)

//go:embed csp-security-policies
var EmbeddedPolicy embed.FS

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

func CreateCISPolicy(fileSystem embed.FS) map[string]string {
	policies := make(map[string]string)

	fs.WalkDir(fileSystem, ".", func(filepath string, info os.DirEntry, err error) error {
		if err != nil {
			logp.Err("Failed to create CIS policy- %+v", err)
			return nil
		}
		if info.IsDir() == false && strings.HasSuffix(info.Name(), ".rego") && !strings.HasSuffix(info.Name(), "test.rego") {

			data, err := fs.ReadFile(fileSystem, filepath)
			if err == nil {
				policies[filepath] = string(data)
			}
		}
		return nil
	})

	return policies
}
