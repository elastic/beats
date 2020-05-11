package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/diff"
)

// RunAndParse runs commands and parse results. Returns map of tool name to check results.
func RunAndParse(ctx context.Context, conf *Config, runners map[string]bool, defaultLevel string, teeMode bool) (*reviewdog.ResultMap, error) {
	var results reviewdog.ResultMap
	// environment variables for each commands
	envs := filteredEnviron()
	cmdBuilder := newCmdBuilder(envs, teeMode)
	var usedRunners []string
	var g errgroup.Group
	semaphoreNum := runtime.NumCPU()
	if teeMode {
		semaphoreNum = 1
	}
	semaphore := make(chan int, semaphoreNum)
	for _, runner := range conf.Runner {
		runner := runner
		if len(runners) != 0 && !runners[runner.Name] {
			continue // Skip this runner.
		}
		usedRunners = append(usedRunners, runner.Name)
		semaphore <- 1
		log.Printf("reviewdog: [start]\trunner=%s", runner.Name)
		fname := runner.Format
		if fname == "" && len(runner.Errorformat) == 0 {
			fname = runner.Name
		}
		opt := &reviewdog.ParserOpt{FormatName: fname, Errorformat: runner.Errorformat}
		p, err := reviewdog.NewParser(opt)
		if err != nil {
			return nil, err
		}
		cmd, stdout, stderr, err := cmdBuilder.build(ctx, runner.Cmd)
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("fail to start command: %v", err)
		}
		g.Go(func() error {
			defer func() { <-semaphore }()
			rs, err := p.Parse(io.MultiReader(stdout, stderr))
			if err != nil {
				return err
			}
			log.Printf("reviewdog: [finish]\trunner=%s", runner.Name)
			level := runner.Level
			if level == "" {
				level = defaultLevel
			}
			results.Store(runner.Name, &reviewdog.Result{Level: level, CheckResults: rs})
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("fail to run reviewdog: %v", err)
	}
	if err := checkUnknownRunner(runners, usedRunners); err != nil {
		return nil, err
	}
	return &results, nil
}

// Run runs reviewdog tasks based on Config.
func Run(ctx context.Context, conf *Config, runners map[string]bool, c reviewdog.CommentService, d reviewdog.DiffService, teeMode bool) error {
	results, err := RunAndParse(ctx, conf, runners, "", teeMode) // Level is not used.
	if err != nil {
		return err
	}
	if results.Len() == 0 {
		return nil
	}
	b, err := d.Diff(ctx)
	if err != nil {
		return err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(b))
	if err != nil {
		return err
	}
	var g errgroup.Group
	results.Range(func(toolname string, result *reviewdog.Result) {
		rs := result.CheckResults
		g.Go(func() error {
			return reviewdog.RunFromResult(ctx, c, rs, filediffs, d.Strip(), toolname)
		})
	})
	return g.Wait()
}

var secretEnvs = [...]string{
	"REVIEWDOG_GITHUB_API_TOKEN",
	"REVIEWDOG_GITLAB_API_TOKEN",
	"REVIEWDOG_TOKEN",
}

func filteredEnviron() []string {
	for _, name := range secretEnvs {
		defer func(name, value string) {
			if value != "" {
				os.Setenv(name, value)
			}
		}(name, os.Getenv(name))
		os.Unsetenv(name)
	}
	return os.Environ()
}

func checkUnknownRunner(specifiedRunners map[string]bool, usedRunners []string) error {
	if len(specifiedRunners) == 0 {
		return nil
	}
	for _, r := range usedRunners {
		delete(specifiedRunners, r)
	}
	var rs []string
	for r := range specifiedRunners {
		rs = append(rs, r)
	}
	if len(specifiedRunners) != 0 {
		return fmt.Errorf("runner not found: [%s]", strings.Join(rs, ","))
	}
	return nil
}
