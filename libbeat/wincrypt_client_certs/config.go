package wincrypt_client_certs

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"go.uber.org/multierr"
	"regexp"
)

type Config struct {
	Stores []string `config:"stores" yaml:"stores,omitempty"` // list of system stores to look for Certificates ["CurrentUser/My", "LocalMachine/My"]
	Query string `config:"query" yaml:"query,omitempty"` // a query to select Certificates
}

var storeDefRegexp = regexp.MustCompile("(?i)(LocalMachine|CurrentUser)/[a-zA-Z]+")

func (c *Config)Validate() error {
	var result error

	for _, storeDef := range(c.Stores) {
        if !storeDefRegexp.MatchString(storeDef) {
            result = multierr.Append(result, fmt.Errorf("wincrypt: invalid store definition %q", storeDef))
		}
	}

	_, err := govaluate.NewEvaluableExpression(c.Query)
	if err != nil {
		result = multierr.Append(result, fmt.Errorf("wincrypt: syntax error in query: %s", err.Error()))
	}

	return result
}
