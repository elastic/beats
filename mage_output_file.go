// +build ignore

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	
)

func main() {
	// Use local types and functions in order to avoid name conflicts with additional magefiles.
	type arguments struct {
		Verbose       bool          // print out log statements
		List          bool          // print out a list of targets
		Help          bool          // print out help for a specific target
		Timeout       time.Duration // set a timeout to running the targets
		Args          []string      // args contain the non-flag command-line arguments
	}

	parseBool := func(env string) bool {
		val := os.Getenv(env)
		if val == "" {
			return false
		}		
		b, err := strconv.ParseBool(val)
		if err != nil {
			log.Printf("warning: environment variable %s is not a valid bool value: %v", env, val)
			return false
		}
		return b
	}

	parseDuration := func(env string) time.Duration {
		val := os.Getenv(env)
		if val == "" {
			return 0
		}		
		d, err := time.ParseDuration(val)
		if err != nil {
			log.Printf("warning: environment variable %s is not a valid duration value: %v", env, val)
			return 0
		}
		return d
	}
	args := arguments{}
	fs := flag.FlagSet{}
	fs.SetOutput(os.Stdout)

	// default flag set with ExitOnError and auto generated PrintDefaults should be sufficient
	fs.BoolVar(&args.Verbose, "v", parseBool("MAGEFILE_VERBOSE"), "show verbose output when running targets")
	fs.BoolVar(&args.List, "l", parseBool("MAGEFILE_LIST"), "list targets for this binary")
	fs.BoolVar(&args.Help, "h", parseBool("MAGEFILE_HELP"), "print out help for a specific target")
	fs.DurationVar(&args.Timeout, "t", parseDuration("MAGEFILE_TIMEOUT"), "timeout in duration parsable format (e.g. 5m30s)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stdout, `
%s [options] [target]

Commands:
  -l    list targets in this binary
  -h    show this help

Options:
  -h    show description of a target
  -t <string>
        timeout in duration parsable format (e.g. 5m30s)
  -v    show verbose output when running targets
 `[1:], filepath.Base(os.Args[0]))
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		// flag will have printed out an error already.
		return
	}
	args.Args = fs.Args()
	if args.Help && len(args.Args) == 0 {
		fs.Usage()
		return
	}
	  
	list := func() error {
		
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
		fmt.Println("Targets:")
			fmt.Fprintln(w, "  dumpVariables\t" + "writes the template variables and values to stdout.")
			fmt.Fprintln(w, "  fmt\t" + "formats code and adds license headers.")
			fmt.Fprintln(w, "  packageBeatDashboards\t" + "packages the dashboards from all Beats into a zip file.")
		err := w.Flush()
		return err
	}

	var ctx context.Context
	var ctxCancel func()

	getContext := func() (context.Context, func()) {
		if ctx != nil {
			return ctx, ctxCancel
		}

		if args.Timeout != 0 {
			ctx, ctxCancel = context.WithTimeout(context.Background(), args.Timeout)
		} else {
			ctx = context.Background()
			ctxCancel = func() {}
		}
		return ctx, ctxCancel
	}

	runTarget := func(fn func(context.Context) error) interface{} {
		var err interface{}
		ctx, cancel := getContext()
		d := make(chan interface{})
		go func() {
			defer func() {
				err := recover()
				d <- err
			}()
			err := fn(ctx)
			d <- err
		}()
		select {
		case <-ctx.Done():
			cancel()
			e := ctx.Err()
			fmt.Printf("ctx err: %v\n", e)
			return e
		case err = <-d:
			cancel()
			return err
		}
	}
	// This is necessary in case there aren't any targets, to avoid an unused
	// variable error.
	_ = runTarget

	handleError := func(logger *log.Logger, err interface{}) {
		if err != nil {
			logger.Printf("Error: %v\n", err)
			type code interface {
				ExitStatus() int
			}
			if c, ok := err.(code); ok {
				os.Exit(c.ExitStatus())
			}
			os.Exit(1)
		}
	}
	_ = handleError

	log.SetFlags(0)
	if !args.Verbose {
		log.SetOutput(ioutil.Discard)
	}
	logger := log.New(os.Stderr, "", 0)
	if args.List {
		if err := list(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		return
	}

	targets := map[string]bool {
		
		"dumpvariables": true,
		"fmt": true,
		"packagebeatdashboards": true,
		
		
	}

	var unknown []string
	for _, arg := range args.Args {
		if !targets[strings.ToLower(arg)] {
			unknown = append(unknown, arg)
		}
	}
	if len(unknown) == 1 {
		logger.Println("Unknown target specified:", unknown[0])
		os.Exit(2)
	}
	if len(unknown) > 1 {
		logger.Println("Unknown targets specified:", strings.Join(unknown, ", "))
		os.Exit(2)
	}

	if args.Help {
		if len(args.Args) < 1 {
			logger.Println("no target specified")
			os.Exit(1)
		}
		switch strings.ToLower(args.Args[0]) {
			case "dumpvariables":
				fmt.Print("mage dumpvariables:\n\n")
				fmt.Println("DumpVariables writes the template variables and values to stdout.")
				fmt.Println()
				
				var aliases []string
				if len(aliases) > 0 {
					fmt.Printf("Aliases: %s\n\n", strings.Join(aliases, ", "))
				}
				return
			case "fmt":
				fmt.Print("mage fmt:\n\n")
				fmt.Println("Fmt formats code and adds license headers.")
				fmt.Println()
				
				var aliases []string
				if len(aliases) > 0 {
					fmt.Printf("Aliases: %s\n\n", strings.Join(aliases, ", "))
				}
				return
			case "packagebeatdashboards":
				fmt.Print("mage packagebeatdashboards:\n\n")
				fmt.Println("PackageBeatDashboards packages the dashboards from all Beats into a zip file. The dashboards must be generated first.")
				fmt.Println()
				
				var aliases []string
				if len(aliases) > 0 {
					fmt.Printf("Aliases: %s\n\n", strings.Join(aliases, ", "))
				}
				return
			
			default:
				logger.Printf("Unknown target: %q\n", args.Args[0])
				os.Exit(1)
		}
	}
	if len(args.Args) < 1 {
		if err := list(); err != nil {
			logger.Println("Error:", err)
			os.Exit(1)
		}
		return
	}
	for _, target := range args.Args {
		switch strings.ToLower(target) {
		
		}
		switch strings.ToLower(target) {
		
			case "dumpvariables":
				if args.Verbose {
					logger.Println("Running target:", "DumpVariables")
				}
							wrapFn := func(ctx context.Context) error {
				return DumpVariables()
			}
			err := runTarget(wrapFn)
				handleError(logger, err)
			case "fmt":
				if args.Verbose {
					logger.Println("Running target:", "Fmt")
				}
							wrapFn := func(ctx context.Context) error {
				Fmt()
				return nil
			}
			err := runTarget(wrapFn)
				handleError(logger, err)
			case "packagebeatdashboards":
				if args.Verbose {
					logger.Println("Running target:", "PackageBeatDashboards")
				}
							wrapFn := func(ctx context.Context) error {
				return PackageBeatDashboards()
			}
			err := runTarget(wrapFn)
				handleError(logger, err)
		
		default:
			// should be impossible since we check this above.
			logger.Printf("Unknown target: %q\n", args.Args[0])
			os.Exit(1)
		}
	}
}




