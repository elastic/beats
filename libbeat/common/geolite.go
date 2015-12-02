package common

import (
	"os"
	"path/filepath"

	"github.com/elastic/libbeat/logp"

	"github.com/nranchev/go-libGeoIP"
)

type Geoip struct {
	Paths *[]string
}

func LoadGeoIPData(config Geoip) *libgeo.GeoIP {

	geoipPaths := []string{}

	if config.Paths != nil {
		geoipPaths = *config.Paths
	}
	if len(geoipPaths) == 0 {
		logp.Info("GeoIP disabled: No paths were set under output.geoip.paths")
		// disabled
		return nil
	}

	// look for the first existing path
	var geoipPath string
	for _, path := range geoipPaths {
		fi, err := os.Lstat(path)
		if err != nil {
			logp.Err("GeoIP path could not be loaded: %s", path)
			continue
		}

		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			// follow symlink
			geoipPath, err = filepath.EvalSymlinks(path)
			if err != nil {
				logp.Warn("Could not load GeoIP data: %s", err.Error())
				return nil
			}
		} else {
			geoipPath = path
		}
		break
	}

	if len(geoipPath) == 0 {
		logp.Warn("Couldn't load GeoIP database")
		return nil
	}

	geoLite, err := libgeo.Load(geoipPath)
	if err != nil {
		logp.Warn("Could not load GeoIP data: %s", err.Error())
	}

	logp.Info("Loaded GeoIP data from: %s", geoipPath)
	return geoLite
}
