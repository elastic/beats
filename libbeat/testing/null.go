package testing

type nullDriver struct{}

// NullDriver does nothing, ignores all output and doesn't die on errors
var NullDriver = &nullDriver{}

func (d *nullDriver) Run(name string, f func(Driver)) {
	f(d)
}

func (d *nullDriver) Info(field, value string) {}

func (d *nullDriver) Warn(field, reason string) {}

func (d *nullDriver) Error(field string, err error) {}

func (d *nullDriver) Fatal(field string, err error) {}

func (d *nullDriver) Result(data string) {}
