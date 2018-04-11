// +build stresstest

package stress_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	// import queue types
	"github.com/elastic/beats/libbeat/publisher/pipeline/stress"
	_ "github.com/elastic/beats/libbeat/publisher/queue/memqueue"
	_ "github.com/elastic/beats/libbeat/publisher/queue/spool"
)

// additional flags
var (
	duration time.Duration // -duration <dur>
)

func init() {
	flag.DurationVar(&duration, "duration", 0, "configure max run duration")
}

func TestPipeline(t *testing.T) {
	genConfigs := collectConfigs(t, "configs/gen/*.yml")
	pipelineConfigs := collectConfigs(t, "configs/pipeline/*.yml")
	outConfigs := collectConfigs(t, "configs/out/*.yml")

	info := beat.Info{
		Beat:     "stresser",
		Version:  "0",
		Name:     "stresser.test",
		Hostname: "stresser.test",
	}

	if duration == 0 {
		duration = 15 * time.Second
	}

	configTest(t, "gen", genConfigs, func(t *testing.T, gen string) {
		configTest(t, "pipeline", pipelineConfigs, func(t *testing.T, pipeline string) {
			configTest(t, "out", outConfigs, func(t *testing.T, out string) {

				if testing.Verbose() {
					start := time.Now()
					fmt.Printf("%v Start stress test %v\n", start.Format(time.RFC3339), t.Name())
					defer func() {
						end := time.Now()
						fmt.Printf("%v Finished stress test %v. Duration=%v\n", end.Format(time.RFC3339), t.Name(), end.Sub(start))
					}()
				}

				config, err := common.LoadFiles(gen, pipeline, out)
				if err != nil {
					t.Fatal(err)
				}

				name := t.Name()
				name = strings.Replace(name, "/", "-", -1)
				name = strings.Replace(name, "\\", "-", -1)

				dir, err := ioutil.TempDir("", "")
				if err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(dir)

				// Merge test info into config object
				config.Merge(map[string]interface{}{
					"test": map[string]interface{}{
						"tmpdir": dir,
						"name":   name,
					},
				})

				// check if the pipeline configuration allows for parallel tests
				onErr := func(err error) {
					t.Error(err)
				}

				if err := stress.RunTests(info, duration, config, onErr); err != nil {
					t.Error("Test failed with:", err)
				}
			})
		})
	})
}

func configTest(t *testing.T, typ string, configs []string, fn func(t *testing.T, config string)) {
	for _, config := range configs {
		config := config
		t.Run(testName(typ, config), func(t *testing.T) {
			t.Parallel()
			fn(t, config)
		})
	}
}

func collectConfigs(t *testing.T, pattern string) []string {
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func testName(typ, path string) string {
	return fmt.Sprintf("%v=%v", typ, filepath.Base(path[:len(path)-4]))
}
