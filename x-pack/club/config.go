package main

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"

	"github.com/elastic/beats/v7/x-pack/club/internal/dirs"
)

// flagsConfig is used for parsing all available CLI flags
type flagsConfig struct {
	Reload            bool
	ConfigFiles       []string
	StrictPermissions bool
	Path              dirs.Project
}

type settings struct {
	ConfigID string `config:"id"`
	Inputs   []inputSettings
	Outputs  map[string]*common.Config
	Path     dirs.Project
	Logging  logp.Config
	Registry kvStoreSettings // XXX: copied from filebeat
	Limits   limitsSettings
	Location string // time zone info
	Manager  agentConfigManagerSettings
}

// dynamicSettings can be updated for via file reloading or external services.
// The app instance expects a complete set of mappings. If a service provides
// delta-updates only (or a subset of Inputs that need to be run), we will need
// to merge the delta updates first, before the dynamicSettings can be applied.
type dynamicSettings struct {
	Inputs  []inputSettings
	Outputs map[string]*common.Config
}

// configure global resource limits to be shared with input managers
type limitsSettings struct {
	// heartbeat monitors scheduled concurrent active operations limit
	Monitors int64
}

func (c *flagsConfig) parseArgs(args []string) error {
	basename := filepath.Base(args[0])
	defaultConfigFile := basename + ".yml"

	flags := flag.NewFlagSet(basename, flag.ContinueOnError)
	c.registerFlags(flags)
	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	if len(c.ConfigFiles) == 0 {
		c.ConfigFiles = []string{defaultConfigFile}
	}
	return nil
}

func (c *flagsConfig) registerFlags(flags *flag.FlagSet) {
	common.StringArrVarFlag(flags, &c.ConfigFiles, "c", "configuration files")
	flags.BoolVar(&c.StrictPermissions, "strict.perms", true, "Strict permission checking on config files")
	flags.BoolVar(&c.Reload, "reload", false, "enable config file reloading")

	registerFlagsPath(flags, &c.Path)
}

func (s *settings) validate() error {
	if _, exists := s.Outputs["default"]; !exists {
		return errors.New("no default output configured")
	}

	for _, inp := range s.Inputs {
		if inp.UseOutput == "" {
			continue
		}
		if _, exist := s.Outputs[inp.UseOutput]; !exist {
			return fmt.Errorf("output '%v' not defined", inp.UseOutput)
		}
	}

	return nil
}

func registerFlagsPath(flags *flag.FlagSet, p *dirs.Project) {
	flags.StringVar(&p.Config, "path.config", "", "Configurations directory to look for config files")
	flags.StringVar(&p.Home, "path.home", "", "Home path")
	flags.StringVar(&p.Data, "path.data", "", "Data path")
	flags.StringVar(&p.Logs, "path.logs", "", "Logs path")
}

func initPaths(ps dirs.Project) (dirs.Project, error) {
	proj, err := dirs.ProjectFrom(ps.Home)
	if err != nil {
		return proj, err
	}

	proj = proj.Update(dirs.Project(ps))

	// configure libbeat globals to help inputs accessing them
	paths.InitPaths(&paths.Path{
		Home:   proj.Home,
		Config: proj.Config,
		Data:   proj.Data,
		Logs:   proj.Logs,
	})

	return proj, nil
}
