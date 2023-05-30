package diagnostics

import (
	"fmt"
	"os"

	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// GetRawFileOrErrorString is a convinence method that will return either the contents of the specified file,
// or the error that results from opening the file
func GetRawFileOrErrorString(res resolve.Resolver, path string) []byte {
	fullPath := res.ResolveHostFS(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return []byte(fmt.Sprintf("Error fetching data from %s: %s", fullPath, err))
	}
	return data
}
