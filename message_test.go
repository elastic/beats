package main

import (
    "testing"
)

func TestMessage_Flags(t *testing.T) {
    if MSG_TYPE_HTTP != 3 {
        t.Error("Bad flag value")
    }

    if HTTP_FLAGS_DIR_INITIAL != 1 {
        t.Error("Bad flag value")
    }

    if HTTP_FLAGS_IS_REQUEST != 2 {
        t.Error("Bad flag value")
    }
}
