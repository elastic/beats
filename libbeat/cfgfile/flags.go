package cfgfile

import (
	"flag"
	"strings"
)

type argList struct {
	list      []string
	isDefault bool
	f         *flag.Flag
}

func flagArgList(name string, def string, usage string) *argList {
	l := &argList{
		list:      []string{def},
		isDefault: true,
	}
	flag.Var(l, name, usage)
	l.f = flag.Lookup(name)
	if l.f == nil {
		panic("Failed to lookup registered flag")
	}
	return l
}

func (l *argList) SetDefault(v string) {
	l.f.DefValue = v
	l.list = []string{v}
	l.isDefault = true
}

func (l *argList) String() string {
	return strings.Join(l.list, ", ")
}

func (l *argList) Set(v string) error {
	if l.isDefault {
		l.list = []string{v}
	} else {
		l.list = append(l.list, v)
	}
	l.isDefault = false
	return nil
}

func (l *argList) Get() interface{} {
	return l.list
}
