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
	// Only update value if we are still in the default
	if l.isDefault {
		l.list = []string{v}
		l.isDefault = true
	}
}

func (l *argList) String() string {
	return strings.Join(l.list, ", ")
}

func (l *argList) Set(v string) error {
	if l.isDefault {
		l.list = []string{v}
	} else {
		// Ignore duplicates, can be caused by multiple flag parses
		for _, f := range l.list {
			if f == v {
				return nil
			}
		}
		l.list = append(l.list, v)
	}
	l.isDefault = false
	return nil
}

func (l *argList) Get() interface{} {
	return l.list
}
