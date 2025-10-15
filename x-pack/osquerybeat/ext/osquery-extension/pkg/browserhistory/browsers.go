// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

// getBrowserPath gets the browser path and handles wildcard expansion
func getBrowserPath(browser string) string {
	if osPaths, exists := pathsByOS[runtime.GOOS]; exists {
		if path, exists := osPaths[browser]; exists {
			return path
		}
	}
	return ""
}

var defaultBrowsers = []string{
	"brave",
	"chrome",
	"chrome_beta",
	"chrome_canary",
	"chrome_dev",
	"chromium",
	"edge",
	"edge_beta",
	"edge_canary",
	"edge_dev",
	"firefox",
	"firefox_beta",
	"firefox_dev",
	"firefox_nightly",
	"onelaunch",
	"opera",
	"opera_gx",
	"safari",
	"safari_tech_preview",
	"vivaldi",
	"wavebrowser",
	"yandex",
}

var pathsByOS = map[string]map[string]string{
	"windows": {
		"brave":         "AppData\\Local\\BraveSoftware\\Brave-Browser",
		"chrome":        "AppData\\Local\\Google\\Chrome",
		"chrome_beta":   "AppData\\Local\\Google\\Chrome Beta",
		"chrome_canary": "AppData\\Local\\Google\\Chrome SxS",
		"chrome_dev":    "AppData\\Local\\Google\\Chrome Dev",
		"chromium":      "AppData\\Local\\Chromium",
		"edge":          "AppData\\Local\\Microsoft\\Edge",
		"edge_beta":     "AppData\\Local\\Microsoft\\Edge Beta",
		"edge_canary":   "AppData\\Local\\Microsoft\\Edge SxS",
		"edge_dev":      "AppData\\Local\\Microsoft\\Edge Dev",
		"firefox":       "AppData\\Roaming\\Mozilla\\Firefox",
		"firefox_dev":   "AppData\\Roaming\\Firefox Developer Edition",
		"onelaunch":     "AppData\\Local\\OneLaunch",
		"opera":         "AppData\\Roaming\\Opera Software\\Opera Stable",
		"opera_gx":      "AppData\\Roaming\\Opera Software\\Opera GX Stable",
		"vivaldi":       "AppData\\Local\\Vivaldi",
		"wavebrowser":   "AppData\\Local\\WaveBrowser",
		"yandex":        "AppData\\Local\\Yandex\\YandexBrowser",
	},
	"macos": {
		"brave":               "Library/Application Support/BraveSoftware/Brave-Browser",
		"chrome":              "Library/Application Support/Google/Chrome",
		"chrome_beta":         "Library/Application Support/Google/Chrome Beta",
		"chrome_canary":       "Library/Application Support/Google/Chrome Canary",
		"chrome_dev":          "Library/Application Support/Google/Chrome Dev",
		"chromium":            "Library/Application Support/Chromium",
		"edge":                "Library/Application Support/Microsoft Edge",
		"edge_beta":           "Library/Application Support/Microsoft Edge Beta",
		"edge_canary":         "Library/Application Support/Microsoft Edge Canary",
		"edge_dev":            "Library/Application Support/Microsoft Edge Dev",
		"firefox":             "Library/Application Support/Firefox",
		"firefox_dev":         "Library/Application Support/Firefox Developer Edition",
		"firefox_nightly":     "Library/Application Support/Firefox Nightly",
		"opera":               "Library/Application Support/com.operasoftware.Opera",
		"opera_gx":            "Library/Application Support/com.operasoftware.OperaGX",
		"safari":              "Library/Safari",
		"safari_tech_preview": "Library/Safari Technology Preview",
		"vivaldi":             "Library/Application Support/Vivaldi",
		"wavebrowser":         "Library/Application Support/WaveBrowser",
		"yandex":              "Library/Application Support/Yandex/YandexBrowser",
	},
	"linux": {
		"brave":       ".config/BraveSoftware/Brave-Browser",
		"chrome":      ".config/google-chrome",
		"chrome_beta": ".config/google-chrome-beta",
		"chrome_dev":  ".config/google-chrome-unstable",
		"chromium":    ".config/chromium",
		"edge":        ".config/microsoft-edge",
		"edge_beta":   ".config/microsoft-edge-beta",
		"edge_dev":    ".config/microsoft-edge-dev",
		"firefox":     ".mozilla/firefox",
		"opera":       ".config/opera",
		"opera_gx":    ".config/opera-gx",
		"vivaldi":     ".config/vivaldi",
		"yandex":      ".config/yandex-browser",
	},
}
