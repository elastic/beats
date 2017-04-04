// Package gotestcover provides multiple packages support for Go test cover.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

var (
	// go build
	flagA    bool
	flagX    bool
	flagRace bool
	flagTags string

	// go test
	flagV            bool
	flagCount        int
	flagCPU          string
	flagParallel     string
	flagRun          string
	flagShort        bool
	flagTimeout      string
	flagCoverMode    string
	flagCoverProfile string

	// custom
	flagParallelPackages = runtime.GOMAXPROCS(0)

	// GAE/Go
	flagGoogleAppEngine bool
)

func main() {
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	err := parseFlags()
	if err != nil {
		return err
	}
	pkgArgs, flagArgs := parseArgs()
	pkgs, err := resolvePackages(pkgArgs)
	if err != nil {
		return err
	}
	cov, failed := runAllPackageTests(pkgs, flagArgs, func(out string) {
		fmt.Print(out)
	})
	err = writeCoverProfile(cov)
	if err != nil {
		return err
	}
	if failed {
		return fmt.Errorf("test failed")
	}
	return nil
}

func parseFlags() error {
	flag.BoolVar(&flagA, "a", flagA, "see 'go build' help")
	flag.BoolVar(&flagX, "x", flagX, "see 'go build' help")
	flag.BoolVar(&flagRace, "race", flagRace, "see 'go build' help")
	flag.StringVar(&flagTags, "tags", flagTags, "see 'go build' help")

	flag.BoolVar(&flagV, "v", flagV, "see 'go test' help")
	flag.IntVar(&flagCount, "count", flagCount, "see 'go test' help")
	flag.StringVar(&flagCPU, "cpu", flagCPU, "see 'go test' help")
	flag.StringVar(&flagParallel, "parallel", flagParallel, "see 'go test' help")
	flag.StringVar(&flagRun, "run", flagRun, "see 'go test' help")
	flag.BoolVar(&flagShort, "short", flagShort, "see 'go test' help")
	flag.StringVar(&flagTimeout, "timeout", flagTimeout, "see 'go test' help")
	flag.StringVar(&flagCoverMode, "covermode", flagCoverMode, "see 'go test' help")
	flag.StringVar(&flagCoverProfile, "coverprofile", flagCoverProfile, "see 'go test' help")

	flag.IntVar(&flagParallelPackages, "parallelpackages", flagParallelPackages, "Number of package test run in parallel")

	flag.BoolVar(&flagGoogleAppEngine, "gae", flagGoogleAppEngine, "Bool of Command exec in GAE/Go")

	flag.Parse()
	if flagCoverProfile == "" {
		return fmt.Errorf("flag coverprofile must be set")
	}
	if flagParallelPackages < 1 {
		return fmt.Errorf("flag parallelpackages must be greater than or equal to 1")
	}
	return nil
}

func parseArgs() (pkgArgs, flagArgs []string) {
	args := flag.Args()
	for i, a := range args {
		if strings.HasPrefix(a, "-") {
			return args[:i], args[i:]
		}
	}
	return args, nil
}

func resolvePackages(pkgArgs []string) ([]string, error) {
	cmdArgs := []string{"list"}
	cmdArgs = append(cmdArgs, pkgArgs...)
	cmdOut, err := runGoCommand(cmdArgs...)
	if err != nil {
		return nil, err
	}
	var pkgs []string
	sc := bufio.NewScanner(bytes.NewReader(cmdOut))
	for sc.Scan() {
		pkgs = append(pkgs, sc.Text())
	}
	return pkgs, nil
}

func runAllPackageTests(pkgs []string, flgs []string, pf func(string)) ([]byte, bool) {
	pkgch := make(chan string)
	type res struct {
		out string
		cov []byte
		err error
	}
	resch := make(chan res)
	wg := new(sync.WaitGroup)
	wg.Add(flagParallelPackages)
	go func() {
		for _, pkg := range pkgs {
			pkgch <- pkg
		}
		close(pkgch)
		wg.Wait()
		close(resch)
	}()
	for i := 0; i < flagParallelPackages; i++ {
		go func() {
			for p := range pkgch {
				out, cov, err := runPackageTests(p, flgs)
				resch <- res{
					out: out,
					cov: cov,
					err: err,
				}
			}
			wg.Done()
		}()
	}
	failed := false
	var cov []byte
	for r := range resch {
		if r.err == nil {
			pf(r.out)
			cov = append(cov, r.cov...)
		} else {
			pf(r.err.Error())
			failed = true
		}
	}
	return cov, failed
}

func runPackageTests(pkg string, flgs []string) (out string, cov []byte, err error) {
	coverFile, err := ioutil.TempFile("", "gotestcover-")
	if err != nil {
		return "", nil, err
	}
	defer os.Remove(coverFile.Name())
	defer coverFile.Close()
	var args []string
	args = append(args, "test")

	if flagA {
		args = append(args, "-a")
	}
	if flagX {
		args = append(args, "-x")
	}
	if flagRace {
		args = append(args, "-race")
	}
	if flagTags != "" {
		args = append(args, "-tags", flagTags)
	}

	if flagV {
		args = append(args, "-v")
	}
	if flagCount != 0 {
		args = append(args, "-count", fmt.Sprint(flagCount))
	}
	if flagCPU != "" {
		args = append(args, "-cpu", flagCPU)
	}
	if flagParallel != "" {
		args = append(args, "-parallel", flagParallel)
	}
	if flagRun != "" {
		args = append(args, "-run", flagRun)
	}
	if flagShort {
		args = append(args, "-short")
	}
	if flagTimeout != "" {
		args = append(args, "-timeout", flagTimeout)
	}
	args = append(args, "-cover")
	if flagCoverMode != "" {
		args = append(args, "-covermode", flagCoverMode)
	}
	args = append(args, "-coverprofile", coverFile.Name())

	args = append(args, pkg)

	args = append(args, flgs...)

	cmdOut, err := runGoCommand(args...)
	if err != nil {
		return "", nil, err
	}
	cov, err = ioutil.ReadAll(coverFile)
	if err != nil {
		return "", nil, err
	}
	cov = removeFirstLine(cov)
	return string(cmdOut), cov, nil
}

func writeCoverProfile(cov []byte) error {
	if len(cov) == 0 {
		return nil
	}
	buf := new(bytes.Buffer)
	mode := flagCoverMode
	if mode == "" {
		if flagRace {
			mode = "atomic"
		} else {
			mode = "set"
		}
	}
	fmt.Fprintf(buf, "mode: %s\n", mode)
	buf.Write(cov)
	return ioutil.WriteFile(flagCoverProfile, buf.Bytes(), os.FileMode(0644))
}

func runGoCommand(args ...string) ([]byte, error) {
	goCmd := "go"
	if flagGoogleAppEngine {
		goCmd = "goapp"
	}
	cmd := exec.Command(goCmd, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command %s: %s\n%s", cmd.Args, err, out)
	}
	return out, nil
}

func removeFirstLine(b []byte) []byte {
	out := new(bytes.Buffer)
	sc := bufio.NewScanner(bytes.NewReader(b))
	firstLine := true
	for sc.Scan() {
		if firstLine {
			firstLine = false
			continue
		}
		fmt.Fprintf(out, "%s\n", sc.Bytes())
	}
	return out.Bytes()
}
