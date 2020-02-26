package ecs

import "github.com/urso/ecslog/fld"

func Message(s string) fld.Field { return ecsString("message", s) }

func Tags(tags ...string) fld.Field { return ecsAny("tags", tags) }

func Labels(labels map[string]string) fld.Field { return ecsAny("labels", labels) }

func (nsLog) Name(name string) fld.Field      { return ecsString("log.name", name) }
func (nsLog) FilePath(s string) fld.Field     { return ecsString("log.file.path", s) }
func (nsLog) FileLine(i int) fld.Field        { return ecsInt("log.file.line", i) }
func (nsLog) FileBasename(s string) fld.Field { return ecsString("log.file.basename", s) }
