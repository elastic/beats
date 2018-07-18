package mage

// var only for tests
var tpl = `// +build ignore

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

func main() {
	// These functions are local variables to avoid name conflicts with 
	// magefiles.
	list := func() error {
		{{- $default := .Default}}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
		fmt.Println("Targets:")
		{{- range .Funcs}}
		fmt.Fprintln(w, "  {{lowerfirst .Name}}{{if eq .Name $default}}*{{end}}\t" + {{printf "%q" .Synopsis}})
		{{- end}}
		err := w.Flush()
		{{- if .Default}}
		if err == nil {
			fmt.Println("\n* default target")
		}
		{{- end}}
		return err
	}

	var ctx context.Context
	var ctxCancel func()

	getContext := func() (context.Context, func()) {
		if ctx != nil {
			return ctx, ctxCancel
		}

		if os.Getenv("MAGEFILE_TIMEOUT") != "" {
			timeout, err := time.ParseDuration(os.Getenv("MAGEFILE_TIMEOUT"))
			if err != nil {
				fmt.Printf("timeout error: %v\n", err)
				os.Exit(1)
			}

			ctx, ctxCancel = context.WithTimeout(context.Background(), timeout)
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
	if os.Getenv("MAGEFILE_VERBOSE") == "" {
		log.SetOutput(ioutil.Discard)
	}
	logger := log.New(os.Stderr, "", 0)
	if os.Getenv("MAGEFILE_LIST") != "" {
		if err := list(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		return
	}

	targets := map[string]bool {
		{{range $alias, $funci := .Aliases}}"{{lower $alias}}": true,
		{{end}}
		{{range .Funcs}}"{{lower .Name}}": true,
		{{end}}
	}

	var unknown []string
	for _, arg := range os.Args[1:] {
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

	if os.Getenv("MAGEFILE_HELP") != "" {
		if len(os.Args) < 2 {
			logger.Println("no target specified")
			os.Exit(1)
		}
		switch strings.ToLower(os.Args[1]) {
			{{range .Funcs}}case "{{lower .Name}}":
				fmt.Print("mage {{lower .Name}}:\n\n")
				{{if ne .Comment ""}}fmt.Println({{printf "%q" .Comment}}){{end}}
				var aliases []string
				{{- $name := .Name -}}
				{{range $alias, $func := $.Aliases}}
				{{if eq $name $func}}aliases = append(aliases, "{{$alias}}"){{end -}}
				{{- end}}
				if len(aliases) > 0 {
					fmt.Printf("Aliases: %s\n\n", strings.Join(aliases, ", "))
				}
				return
			{{end}}
			default:
				logger.Printf("Unknown target: %q\n", os.Args[1])
				os.Exit(1)
		}	
	}

	if len(os.Args) < 2 {
	{{- if .Default}}
		{{.DefaultFunc.TemplateString}}
		handleError(logger, err)
		return
	{{- else}}
		if err := list(); err != nil {
			logger.Println("Error:", err)
			os.Exit(1)
		}
		return
	{{- end}}
	}
	for _, target := range os.Args[1:] {
		switch strings.ToLower(target) {
		{{range $alias, $func := .Aliases}}
		case "{{lower $alias}}":
			target = "{{$func}}"
		{{- end}}
		}
		switch strings.ToLower(target) {
		{{range .Funcs }}
		case "{{lower .Name}}":
			if os.Getenv("MAGEFILE_VERBOSE") != "" {
				logger.Println("Running target:", "{{.Name}}")
			}
			{{.TemplateString}}
			handleError(logger, err)
		{{- end}}
		default:
			// should be impossible since we check this above.
			logger.Printf("Unknown target: %q\n", os.Args[1])
			os.Exit(1)
		}
	}
}




`
