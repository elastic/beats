package outputs

import (
	"os"
	"path/filepath"

	"github.com/elastic/infrabeat/common"
	"github.com/elastic/infrabeat/logp"

	"github.com/nranchev/go-libGeoIP"
)

var _GeoLite *libgeo.GeoIP

type GeoIPConfig struct {
	Geoip Geoip
}

type Geoip struct {
	Paths []string
}

func LoadGeoIPData(cfg common.Config) error {

	var config GeoIPConfig

	err := common.DecodeConfig(cfg, &config)
	if err != nil {
		return err
	}

	geoip_paths := []string{
		"/usr/share/GeoIP/GeoIP.dat",
		"/usr/local/var/GeoIP/GeoIP.dat",
	}
	if cfg.Meta.IsDefined("geoip", "paths") {
		geoip_paths = config.Geoip.Paths
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

	_GeoLite, err = libgeo.Load(geoip_path)
	if err != nil {
		logp.Warn("Could not load GeoIP data: %s", err.Error())
	}

	logp.Info("Loaded GeoIP data from: %s", geoip_path)
	return nil
}
