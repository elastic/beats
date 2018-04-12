package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	logpcfg "github.com/elastic/beats/libbeat/logp/configure"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/publisher/pipeline/stress"
	"github.com/elastic/beats/libbeat/service"

	// import queue types
	_ "github.com/elastic/beats/libbeat/publisher/queue/memqueue"
	_ "github.com/elastic/beats/libbeat/publisher/queue/spool"

	// import outputs
	_ "github.com/elastic/beats/libbeat/outputs/console"
	_ "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/libbeat/outputs/fileout"
	_ "github.com/elastic/beats/libbeat/outputs/logstash"
)

var (
	duration   time.Duration // -duration <duration>
	overwrites = common.SettingFlag(nil, "E", "Configuration overwrite")
)

type config struct {
	Path    paths.Path
	Logging *common.Config
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	info := beat.Info{
		Beat:     "stresser",
		Version:  "0",
		Name:     "stresser.test",
		Hostname: "stresser.test",
	}

	flag.DurationVar(&duration, "duration", 0, "Test duration (default 0)")
	flag.Parse()

	files := flag.Args()
	fmt.Println("load config files:", files)

	cfg, err := common.LoadFiles(files...)
	if err != nil {
		return err
	}

	service.BeforeRun()
	defer service.Cleanup()

	if err := cfg.Merge(overwrites); err != nil {
		return err
	}

	config := config{}
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	if err := paths.InitPaths(&config.Path); err != nil {
		return err
	}
	if err = logpcfg.Logging("test", config.Logging); err != nil {
		return err
	}

	cfg.PrintDebugf("input config:")

	return stress.RunTests(info, duration, cfg, nil)
}

func startHTTP(bind string) {
	go func() {
		http.ListenAndServe(bind, nil)
	}()
}
