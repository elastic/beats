// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"fmt"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"
)

// extractBrowserMonitorParams removes "params" from every browser monitor in
// beatconfig and pre-parses each with PathSep("") to keep dotted keys literal.
// Returns a map from monitor index to the pre-parsed params. It mutates beatconfig.
func extractBrowserMonitorParams(beatconfig map[string]any) (map[int]*conf.C, error) {
	heartbeat, _ := beatconfig["heartbeat"].(map[string]any)
	monitors, _ := heartbeat["monitors"].([]any)
	extracted := make(map[int]*conf.C)
	for i, m := range monitors {
		monitor, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if kind, _ := monitor["type"].(string); kind != "browser" {
			continue
		}
		params, ok := monitor["params"].(map[string]any)
		if !ok {
			continue
		}
		parsed, err := ucfg.NewFrom(params, ucfg.PathSep(""))
		if err != nil {
			return nil, fmt.Errorf("monitor %d: parsing browser params: %w", i, err)
		}
		delete(monitor, "params")
		extracted[i] = (*conf.C)(parsed)
	}
	return extracted, nil
}

// restoreBrowserMonitorParams puts the extracted browser params back into
// rawConfig, preserving literal dotted keys.
func restoreBrowserMonitorParams(rawConfig *conf.C, params map[int]*conf.C) error {
	if len(params) == 0 {
		return nil
	}
	hbCfg, err := rawConfig.Child("heartbeat", -1)
	if err != nil {
		return fmt.Errorf("error accessing heartbeat config: %w", err)
	}
	for i, p := range params {
		monitorCfg, err := hbCfg.Child("monitors", i)
		if err != nil {
			return fmt.Errorf("error accessing monitor %d config: %w", i, err)
		}
		if err := monitorCfg.SetChild("params", -1, p); err != nil {
			return fmt.Errorf("monitor %d: restoring browser params: %w", i, err)
		}
	}
	return nil
}
