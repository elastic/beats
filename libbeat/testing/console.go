package testing

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// ConsoleDriver outputs test result to the given stdout/stderr descriptors
type ConsoleDriver struct {
	Stdout   io.Writer
	level    int
	reported bool
	killer   func()
}

// NewConsoleDriver initializes and returns a new console driver with output to given file
func NewConsoleDriver(stdout io.Writer) *ConsoleDriver {
	return NewConsoleDriverWithKiller(stdout, func() { os.Exit(1) })
}

// NewConsoleDriverWithKiller initializes and returns a new console driver with output file
// Killer function will be called on fatal errors
func NewConsoleDriverWithKiller(stdout io.Writer, killer func()) *ConsoleDriver {
	return &ConsoleDriver{
		Stdout: stdout,
		level:  0,
		killer: killer,
	}
}

func (d *ConsoleDriver) Run(name string, f func(Driver)) {
	d.printf("%s...\n", name)

	// Run sub func
	driver := &ConsoleDriver{
		Stdout: d.Stdout,
		level:  d.level + 1,
		killer: d.killer,
	}
	f(driver)

	if !driver.reported {
		driver.ok()
	}
}

func (d *ConsoleDriver) Info(field, value string) {
	d.printf("%s: %s\n", field, value)
	d.reported = true
}

func (d *ConsoleDriver) Warn(field, reason string) {
	d.printf("%s... ", field)
	d.warn(reason)
}

func (d *ConsoleDriver) Error(field string, err error) {
	d.printf("%s... ", field)
	if err == nil {
		d.ok()
		return
	}
	d.error(err)
}

func (d *ConsoleDriver) Fatal(field string, err error) {
	d.printf("%s... ", field)
	if err == nil {
		d.ok()
		return
	}
	d.error(err)
	d.killer()
}

func (d *ConsoleDriver) ok() {
	color.New(color.FgGreen).Fprintf(d.Stdout, "OK\n")
	d.reported = true
}

func (d *ConsoleDriver) error(err error) {
	color.New(color.FgRed).Fprintf(d.Stdout, "ERROR ")
	fmt.Fprintln(d.Stdout, err.Error())
	d.reported = true
}

func (d *ConsoleDriver) warn(reason string) {
	color.New(color.FgYellow).Fprintf(d.Stdout, "WARN ")
	fmt.Fprintln(d.Stdout, reason)
	d.reported = true
}

func (d *ConsoleDriver) printf(format string, args ...interface{}) {
	for i := 0; i < d.level; i++ {
		fmt.Fprint(d.Stdout, "  ")
	}
	fmt.Fprintf(d.Stdout, format, args...)
}
