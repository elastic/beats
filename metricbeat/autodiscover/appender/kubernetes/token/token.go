package token

import (
	"fmt"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

func init() {
	autodiscover.Registry.AddAppender("kubernetes.token", NewTokenAppender)
}

type tokenAppender struct {
	TokenPath string
	Condition *processors.Condition
}

// NewTokenAppender creates a token appender that can append a bearer token required to authenticate with
// protected endpoints
func NewTokenAppender(cfg *common.Config) (autodiscover.Appender, error) {
	cfgwarn.Beta("The token appender is beta")
	conf := defaultConfig()

	err := cfg.Unpack(&conf)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack config due to error: %v", err)
	}

	// Attempt to create a condition. If fails then report error
	cond, err := processors.NewCondition(conf.ConditionConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create condition due to error: %v", err)
	}
	appender := tokenAppender{
		TokenPath: conf.TokenPath,
		Condition: cond,
	}

	return &appender, nil
}

// Append picks up a token from a file and adds it to the headers.Authorization section of the metricbeat module
func (t *tokenAppender) Append(event bus.Event) {
	cfgsRaw, ok := event["config"]
	// There are no configs
	if !ok {
		return
	}

	cfgs, ok := cfgsRaw.([]*common.Config)
	// Config key doesnt have an array of config objects
	if !ok {
		return
	}

	// Check if the condition is met. Attempt to append only if that is the case.
	if t.Condition == nil || t.Condition.Check(common.MapStr(event)) == true {
		tok := t.getAuthHeaderFromToken()
		// If token is empty then just return
		if tok == "" {
			return
		}
		for i := 0; i < len(cfgs); i++ {
			// Unpack the config
			cfg := cfgs[i]
			c := common.MapStr{}
			err := cfg.Unpack(&c)
			if err != nil {
				logp.Debug("kubernetes.config", "unable to unpack config due to error: %v", err)
				continue
			}
			var headers common.MapStr
			if hRaw, ok := c["headers"]; ok {
				// If headers is not a map then continue to next config
				if headers, ok = hRaw.(common.MapStr); !ok {
					continue
				}
			} else {
				headers = common.MapStr{}
			}

			// Assign authorization header and add it back to the config
			headers["Authorization"] = tok
			c["headers"] = headers

			// Repack the configuration
			newCfg, err := common.NewConfigFrom(&c)
			if err != nil {
				logp.Debug("kubernetes.config", "unable to repack config due to error: %v", err)
				continue
			}
			cfgs[i] = newCfg
		}

		event["config"] = cfgs
	}
}

func (t *tokenAppender) getAuthHeaderFromToken() string {
	var token string

	if t.TokenPath != "" {
		b, err := ioutil.ReadFile(t.TokenPath)
		if err != nil {
			logp.Err("Reading token file failed with err: %v", err)
		}

		if len(b) != 0 {
			if b[len(b)-1] == '\n' {
				b = b[0 : len(b)-1]
			}
			token = fmt.Sprintf("Bearer %s", string(b))
		}
	}

	return token
}
