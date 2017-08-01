package beat

import "errors"

// GracefulExit is an error that signals to exit with a code of 0.
var GracefulExit = errors.New("graceful exit")
