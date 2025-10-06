// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
)

// getBrowserPath gets the browser path and handles wildcard expansion
func getBrowserPath(browser string) string {
	osName := runtime.GOOS
	switch osName {
	case "windows", "linux":
	case "darwin":
		osName = "macos"
	default:
		return ""
	}
	if osPaths, exists := pathsByOS[osName]; exists {
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
	"opera",
	"opera_gx",
	"safari",
	"safari_beta",
	"safari_tech_preview",
	"vivaldi",
	"yandex",
}

var pathsByOS = map[string]map[string]string{
	"windows": {
		"brave":           "AppData\\Local\\BraveSoftware\\Brave-Browser\\User Data\\{profile}\\History",
		"chrome":          "AppData\\Local\\Google\\Chrome\\User Data\\{profile}\\History",
		"chrome_beta":     "AppData\\Local\\Google\\Chrome Beta\\User Data\\{profile}\\History",
		"chrome_canary":   "AppData\\Local\\Google\\Chrome SxS\\User Data\\{profile}\\History",
		"chrome_dev":      "AppData\\Local\\Google\\Chrome Dev\\User Data\\{profile}\\History",
		"chromium":        "AppData\\Local\\Chromium\\User Data\\{profile}\\History",
		"edge":            "AppData\\Local\\Microsoft\\Edge\\User Data\\{profile}\\History",
		"edge_beta":       "AppData\\Local\\Microsoft\\Edge Beta\\User Data\\{profile}\\History",
		"edge_canary":     "AppData\\Local\\Microsoft\\Edge SxS\\User Data\\{profile}\\History",
		"edge_dev":        "AppData\\Local\\Microsoft\\Edge Dev\\User Data\\{profile}\\History",
		"firefox":         "AppData\\Roaming\\Mozilla\\Firefox\\Profiles\\*\\places.sqlite",
		"firefox_beta":    "AppData\\Roaming\\Mozilla\\Firefox\\Profiles\\*\\places.sqlite",
		"firefox_dev":     "AppData\\Roaming\\Firefox Developer Edition\\Profiles\\*\\places.sqlite",
		"firefox_nightly": "AppData\\Roaming\\Mozilla\\Firefox\\Profiles\\*\\places.sqlite",
		"onelaunch":       "AppData\\Local\\OneLaunch\\User Data\\{profile}\\History",
		"opera":           "AppData\\Roaming\\Opera Software\\Opera Stable\\{profile}\\History",
		"opera_gx":        "AppData\\Roaming\\Opera Software\\Opera GX Stable\\{profile}\\History",
		"vivaldi":         "AppData\\Local\\Vivaldi\\User Data\\{profile}\\History",
		"wavebrowser":     "AppData\\Local\\WaveBrowser\\User Data\\{profile}\\History",
		"yandex":          "AppData\\Local\\Yandex\\YandexBrowser\\User Data\\{profile}\\History",
	},
	"macos": {
		"brave":               "Library/Application Support/BraveSoftware/Brave-Browser/{profile}/History",
		"chrome":              "Library/Application Support/Google/Chrome/{profile}/History",
		"chrome_beta":         "Library/Application Support/Google/Chrome Beta/{profile}/History",
		"chrome_canary":       "Library/Application Support/Google/Chrome Canary/{profile}/History",
		"chrome_dev":          "Library/Application Support/Google/Chrome Dev/{profile}/History",
		"chromium":            "Library/Application Support/Chromium/{profile}/History",
		"edge":                "Library/Application Support/Microsoft Edge/{profile}/History",
		"edge_beta":           "Library/Application Support/Microsoft Edge Beta/{profile}/History",
		"edge_canary":         "Library/Application Support/Microsoft Edge Canary/{profile}/History",
		"edge_dev":            "Library/Application Support/Microsoft Edge Dev/{profile}/History",
		"firefox":             "Library/Application Support/Firefox/Profiles/*/places.sqlite",
		"firefox_beta":        "Library/Application Support/Firefox/Profiles/*/places.sqlite",
		"firefox_dev":         "Library/Application Support/Firefox Developer Edition/Profiles/*/places.sqlite",
		"firefox_nightly":     "Library/Application Support/Firefox Nightly/Profiles/*/places.sqlite",
		"opera":               "Library/Application Support/com.operasoftware.Opera/{profile}/History",
		"opera_gx":            "Library/Application Support/com.operasoftware.OperaGX/{profile}/History",
		"safari":              "Library/Safari/History.db",
		"safari_beta":         "Library/Safari/History.db",
		"safari_tech_preview": "Library/Safari Technology Preview/History.db",
		"vivaldi":             "Library/Application Support/Vivaldi/{profile}/History",
		"wavebrowser":         "Library/Application Support/WaveBrowser/{profile}/History",
		"yandex":              "Library/Application Support/Yandex/YandexBrowser/{profile}/History",
	},
	"linux": {
		"brave":           ".config/BraveSoftware/Brave-Browser/{profile}/History",
		"chrome":          ".config/google-chrome/{profile}/History",
		"chrome_beta":     ".config/google-chrome-beta/{profile}/History",
		"chrome_dev":      ".config/google-chrome-unstable/{profile}/History",
		"chromium":        ".config/chromium/{profile}/History",
		"edge":            ".config/microsoft-edge/{profile}/History",
		"edge_beta":       ".config/microsoft-edge-beta/{profile}/History",
		"edge_dev":        ".config/microsoft-edge-dev/{profile}/History",
		"firefox":         ".mozilla/firefox/*/places.sqlite",
		"firefox_beta":    ".mozilla/firefox/*/places.sqlite",
		"firefox_dev":     ".mozilla/firefox/*/places.sqlite",
		"firefox_nightly": ".mozilla/firefox/*/places.sqlite",
		"opera":           ".config/opera/{profile}/History",
		"opera_gx":        ".config/opera-gx/{profile}/History",
		"vivaldi":         ".config/vivaldi/{profile}/History",
		"yandex":          ".config/yandex-browser/{profile}/History",
	},
}

var browserParsers = map[string]func(ctx context.Context, queryContext table.QueryContext, browser string, paths string, log func(m string, kvs ...any)) ([]map[string]string, error){
	"brave":               chromiumParser,
	"chrome":              chromiumParser,
	"chrome_beta":         chromiumParser,
	"chrome_canary":       chromiumParser,
	"chrome_dev":          chromiumParser,
	"chromium":            chromiumParser,
	"edge":                chromiumParser,
	"edge_beta":           chromiumParser,
	"edge_canary":         chromiumParser,
	"edge_dev":            chromiumParser,
	"opera":               chromiumParser,
	"opera_gx":            chromiumParser,
	"vivaldi":             chromiumParser,
	"yandex":              chromiumParser,
	"firefox":             nil, // TODO: implement Firefox parser
	"firefox_beta":        nil, // TODO: implement Firefox parser
	"firefox_dev":         nil, // TODO: implement Firefox parser
	"firefox_nightly":     nil, // TODO: implement Firefox parser
	"safari":              nil, // TODO: implement Safari parser
	"safari_beta":         nil, // TODO: implement Safari parser
	"safari_tech_preview": nil, // TODO: implement Safari parser
}
