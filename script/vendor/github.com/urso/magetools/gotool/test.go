package gotool

import (
	"context"
	"os"
	"strings"

	"github.com/urso/magetools/clitool"
)

type GoTest func(context context.Context, opts ...clitool.ArgOpt) error

type goTest struct {
	g *Go
}

func makeTest(g *Go) GoTest {
	gt := &goTest{g}
	return gt.Do
}

func (GoTest) WithCoverage(to string) clitool.ArgOpt {
	return clitool.Combine(clitool.Flag("-cover", ""), clitool.FlagIf("-test.coverprofile", to))
}

func (GoTest) Short(b bool) clitool.ArgOpt          { return clitool.BoolFlag("-test.short", b) }
func (GoTest) UseBinary(path string) clitool.ArgOpt { return clitool.ExtraIf("use", path) }
func (GoTest) UseBinaryIf(path string, b bool) clitool.ArgOpt {
	return clitool.When(b, clitool.ExtraIf("use", path))
}
func (GoTest) OS(os string) clitool.ArgOpt        { return clitool.EnvIf("GOOS", os) }
func (GoTest) ARCH(os string) clitool.ArgOpt      { return clitool.EnvIf("GOARCH", os) }
func (GoTest) Create(b bool) clitool.ArgOpt       { return clitool.BoolFlag("-c", b) }
func (GoTest) Out(path string) clitool.ArgOpt     { return clitool.FlagIf("-o", path) }
func (GoTest) Package(path string) clitool.ArgOpt { return clitool.Positional(path) }
func (GoTest) Verbose(b bool) clitool.ArgOpt      { return clitool.BoolFlag("-test.v", b) }

func (gt *goTest) Do(context context.Context, opts ...clitool.ArgOpt) error {
	args := clitool.CreateArgs(opts...)

	if bin := args.GetExtra("use"); bin != "" {
		var flags []clitool.CommandFlag
		for _, f := range args.Flags {
			if strings.HasPrefix(f.Key, "-test.") {
				flags = append(flags, f)
			}
		}

		args.Flags = flags
		return gt.g.Exec(context, bin, args, os.Stdout, os.Stderr)
	}

	return gt.g.ExecGo(context, []string{"test"}, args, os.Stdout, os.Stderr)
}
