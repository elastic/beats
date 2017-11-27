package testing

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

// ConsoleDriver outputs test result to the given stdout/stderr descriptors
type ConsoleDriver struct {
	Stdout   io.Writer
	level    int
	reported bool
	killer   func()
	result   string
}

// NewConsoleDriver initializes and returns a new console driver with output to given file
func NewConsoleDriver(stdout io.Writer) *ConsoleDriver {
	return NewConsoleDriverWithKiller(stdout, func() { os.Exit(1) })
}

// NewConsoleDriverWithKiller initializes and returns a new console driver with output file
// Killer function will be called on fatal errors
func NewConsoleDriverWithKiller(stdout io.Writer, killer func()) *ConsoleDriver {
	// On Windows we must wrap file outputs with a Colorable to achieve the right
	// escape sequences.
	if f, ok := stdout.(*os.File); ok && runtime.GOOS == "windows" {
		stdout = colorable.NewColorable(f)
	}
	return &ConsoleDriver{
		Stdout:   stdout,
		level:    0,
		killer:   killer,
		reported: true,
	}
}

func (d *ConsoleDriver) Run(name string, f func(Driver)) {
	if !d.reported {
		fmt.Fprintln(d.Stdout, "")
	}
	d.printf("%s...", name)

	// Run sub func
	driver := &ConsoleDriver{
		Stdout: d.Stdout,
		level:  d.level + 1,
		killer: d.killer,
	}
	f(driver)

	if !driver.reported {
		color.New(color.FgGreen).Fprintf(driver.Stdout, "OK\n")
		driver.reported = true
	}

	if driver.result != "" {
		driver.Info("result", driver.indent(driver.result))
	}

	d.reported = true
}

func (d *ConsoleDriver) Info(field, value string) {
	if !d.reported {
		fmt.Fprintln(d.Stdout, "")
	}
	d.printf("%s: %s\n", field, value)
	d.reported = true
}

func (d *ConsoleDriver) Warn(field, reason string) {
	if !d.reported {
		fmt.Fprintln(d.Stdout, "")
	}
	d.printf("%s... ", field)
	color.New(color.FgYellow).Fprintf(d.Stdout, "WARN ")
	fmt.Fprintln(d.Stdout, reason)
	d.reported = true
}

func (d *ConsoleDriver) Error(field string, err error) {
	if err == nil {
		d.ok(field)
		return
	}
	d.error(field, err)
}

func (d *ConsoleDriver) Fatal(field string, err error) {
	if err == nil {
		d.ok(field)
		return
	}
	d.error(field, err)
	d.killer()
}

func (d *ConsoleDriver) Result(data string) {
	d.result = data
}

func (d *ConsoleDriver) ok(field string) {
	if !d.reported {
		fmt.Fprintln(d.Stdout, "")
	}
	d.printf("%s... ", field)
	color.New(color.FgGreen).Fprintf(d.Stdout, "OK\n")
	d.reported = true
}

func (d *ConsoleDriver) error(field string, err error) {
	if !d.reported {
		fmt.Fprintln(d.Stdout, "")
	}
	d.printf("%s... ", field)
	color.New(color.FgRed).Fprintf(d.Stdout, "ERROR ")
	fmt.Fprintln(d.Stdout, err.Error())
	d.reported = true
}

func (d *ConsoleDriver) printf(format string, args ...interface{}) {
	for i := 0; i < d.level; i++ {
		fmt.Fprint(d.Stdout, "  ")
	}
	fmt.Fprintf(d.Stdout, format, args...)
}

func (d *ConsoleDriver) indent(data string) string {
	res := "\n"
	for _, line := range strings.Split(data, "\n") {
		res += strings.Repeat(" ", d.level+2) + line + "\n"
	}
	return res
}
