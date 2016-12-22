package socket

import (
	"os/user"
	"strconv"
)

// UserCache is a cache of UID to username.
type UserCache map[int]string

// NewUserCache returns a new UserCache.
func NewUserCache() UserCache {
	return map[int]string{0: "root"}
}

// LookupUID looks up a UID and returns the username associated with it. If
// no username could be found an empty string is returned. The value will be
// cached forever.
func (c UserCache) LookupUID(uid int) string {
	if username, found := c[uid]; found {
		return username
	}

	// Cache the value (even on error).
	username, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		c[uid] = ""
		return ""
	}
	c[uid] = username.Name
	return username.Name
}
