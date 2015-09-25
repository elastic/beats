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

	geoip_paths := []string{
		"/usr/share/GeoIP/GeoLiteCity.dat",
		"/usr/local/var/GeoIP/GeoLiteCity.dat",
	}
	if config.Paths != nil {
		geoip_paths = *config.Paths
	}
	if len(geoip_paths) == 0 {
		// disabled
		return nil
	}

	// look for the first existing path
	var geoip_path string
	for _, path := range geoip_paths {
		fi, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			// follow symlink
			geoip_path, err = filepath.EvalSymlinks(path)
			if err != nil {
				logp.Warn("Could not load GeoIP data: %s", err.Error())
				return nil
			}
		} else {
			geoip_path = path
		}
		break
	}

	if len(geoip_path) == 0 {
		logp.Warn("Couldn't load GeoIP database")
		return nil
	}

	geoLite, err := libgeo.Load(geoip_path)
	if err != nil {
		logp.Warn("Could not load GeoIP data: %s", err.Error())
	}

	logp.Info("Loaded GeoIP data from: %s", geoip_path)
	return geoLite
}
