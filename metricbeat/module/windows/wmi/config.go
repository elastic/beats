// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package wmi

import (
	"time"

	wmiquery "github.com/microsoft/wmi/pkg/base/query"
)

type WmibeatConfig struct {
	Period         time.Duration `config:"period"`
	IncludeQueries bool          `config:"include_queries"` // Whether to include the query in the document
	IncludeNull    bool          `config:"include_null"`    // Whether to include or not nil properties
	Host           string        `config:"host"`
	User           string        `config:"username"`
	Password       string        `config:"password"`
	Namespace      string        `config:"namespace"` // Namespace for the queries
	Queries        []QueryConfig `config:"queries"`
}

type QueryConfig struct {
	Query  *wmiquery.WmiQuery
	Class  string   `config:"class"`
	Fields []string `config:"fields"`
	Where  []string `config:"where"`
}

func NewDefaultConfig() WmibeatConfig {
	return WmibeatConfig{
		Period:         10 * time.Second,
		IncludeQueries: false,
		IncludeNull:    false,
		Namespace:      WMIDefaultNamespace,
	}
}
