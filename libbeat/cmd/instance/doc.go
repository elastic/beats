/*
Package instance provides the functions required to manage the life-cycle of a Beat.
It provides the standard mechanism for launching a Beat. It manages
configuration, logging, and publisher initialization and registers a signal
handler to gracefully stop the process.

Each Beat implementation must implement the `Beater` interface and a `Creator`
to create and initialize the Beater instance. See the `Beater` interface and `Creator`
documentation for more details.

To use this package, create a simple main that invokes the Run() function.

  func main() {
  	if err := beat.Run("mybeat", myVersion, beater.New); err != nil {
  		os.Exit(1)
  	}
  }

In the example above, the beater package contains the implementation of the
Beater interface and the New method returns a new instance of Beater. The
Beater implementation is placed into its own package so that it can be reused
or combined with other Beats.

Recommendations

  * Use the logp package for logging rather than writing to stdout or stderr.
  * Do not call os.Exit in any of your code. Return an error instead. Or if your
    code needs to exit without an error, return beat.GracefulExit.
*/
package instance
