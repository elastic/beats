package main

import (
    "syscall"
)

func DropPrivileges() error {
    var err error

    if !_ConfigMeta.IsDefined("runoptions", "uid") {
        // not found, no dropping privileges but no err
        return nil
    }

    return MsgError("Dropping privileges is not supported on Windows")
}
