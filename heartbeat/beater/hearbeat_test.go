package beater

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestBasic(t *testing.T) {
	RunOnce(t, mapstr.M{})
}

func RunOnce(t *testing.T, conf mapstr.M) (err error) {
	c := conf.Clone()
	c["run_once"] = true

	hb, bbeat := MakeHb(t, conf)
	err = hb.Run(bbeat)
	return err
}

func MakeHb(t *testing.T, conf mapstr.M) (beat.Beater, *beat.Beat) {
	instanceBeat, err := instance.NewBeat("heartbeat", "", "", true)
	require.NoError(t, err)
	instanceBeat.RawConfig, err = config.NewConfigFrom(conf)
	require.NoError(t, err)

	err = instanceBeat.Setup(
		instance.Settings{
			Name: "heartbeat",
		},
		New,
		instance.SetupSettings{},
	)
	require.NoError(t, err)

	beatBeat := &instanceBeat.Beat
	hb, err := New(beatBeat, instanceBeat.RawConfig)
	require.NoError(t, err)

	return hb, beatBeat
}

type RunSpec struct {
	flags []*pflag.Flag
}

func BuildDir() string {
	_, b, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(b), "build")
}

func testBuildDir(t *testing.T) string {
	name := t.Name()
	fsName := regexp.MustCompile(`^[A-Za-z0-9]`).ReplaceAllString(strings.ToLower(name), "_")
	return path.Join(BuildDir(), fsName)
}

func RunBeatTest(t *testing.T, conf mapstr.M) {
	buildDir := testBuildDir(t)
	err := os.MkdirAll(buildDir, 0755)
	require.NoError(t, err)

	/*
		outputDir := path.Join(buildDir, "output")
		configDir := path.Join(buildDir, "config")
		dataDir := path.Join(buildDir, "data")

		flags := pflag.NewFlagSet("flags", pflag.ContinueOnError)
		flags.AddFlag(pflag.String("--"))
		flags.AddFlag(pflag.String())

		instance.NewInitializedBeat(instance.Settings{
			RunFlags: flags,
		})
		instance.NewBeat("heartbeat")
	*/
}
