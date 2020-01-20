package gotool

import (
	"context"
	"os"
	"strings"

	"github.com/urso/magetools/clitool"
)

type GoBuild func(context context.Context, opts ...clitool.ArgOpt) error

type goBuild struct {
	g *Go
}

type BuildMode string

const (
	BuildArchive  BuildMode = "archive"
	BuildCArchive BuildMode = "c-archive"
	BuildCShared  BuildMode = "c-shared"
	BuildShared   BuildMode = "shared"
	BuildExe      BuildMode = "exe"
	BuildPIE      BuildMode = "pie"
	BuildPlugin   BuildMode = "plugin"
)

func makeBuild(g *Go) GoBuild {
	gb := &goBuild{g}
	return gb.Do
}

func (gb *goBuild) Do(context context.Context, opts ...clitool.ArgOpt) error {
	return gb.g.ExecGo(context, []string{"build"}, clitool.CreateArgs(opts...), os.Stdout, os.Stderr)
}

func (GoBuild) OS(os string) clitool.ArgOpt            { return clitool.EnvIf("GOOS", os) }
func (GoBuild) ARCH(os string) clitool.ArgOpt          { return clitool.EnvIf("GOARCH", os) }
func (GoBuild) Packages(pkgs ...string) clitool.ArgOpt { return clitool.Positional(pkgs...) }
func (GoBuild) ForceRebuild(b bool) clitool.ArgOpt     { return clitool.BoolFlag("-a", b) }
func (GoBuild) RaceDetector(b bool) clitool.ArgOpt     { return clitool.BoolFlag("-race", b) }
func (GoBuild) Verbose(b bool) clitool.ArgOpt          { return clitool.BoolFlag("-v", b) }
func (GoBuild) Mode(mode BuildMode) clitool.ArgOpt     { return clitool.FlagIf("-buildmode", mode.String()) }
func (GoBuild) GccGoFlags(flags string) clitool.ArgOpt { return clitool.FlagIf("-gccgoflags", flags) }
func (GoBuild) GcFlags(flags string) clitool.ArgOpt    { return clitool.FlagIf("-goflags", flags) }
func (GoBuild) LdFlags(flags string) clitool.ArgOpt    { return clitool.FlagIf("-ldflags", flags) }
func (GoBuild) LinkShared(b bool) clitool.ArgOpt       { return clitool.BoolFlag("-linkshared", b) }
func (GoBuild) Tags(tags ...string) clitool.ArgOpt {
	return clitool.FlagIf("-tags", strings.Join(tags, " "))
}

func (b BuildMode) String() string {
	if b == "" {
		return "<unknown>"
	}
	return string(b)
}
