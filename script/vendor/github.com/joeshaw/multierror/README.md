# multierror #

multierror is a simple Go package for combining multiple `error`s.
This is handy if you are concurrently running operations within
a function that returns only a single `error`.

## API ##

multierror exposes two types.

`multierror.Errors` is a `[]error` with a receiver method `Err()`,
which returns a `multierror.MultiError` instance or `nil`.  You use
this type to collect your errors by appending to it.

`multierror.MultiError` implements the `error` interface.  Its
`Errors` field contains the `multierror.Errors` you originally
constructed.

## Example ##

```go
package main

import (
	"fmt"
	"github.com/joeshaw/multierror"
)

func main() {
	// Collect multiple errors together in multierror.Errors
	var e1 multierror.Errors
	e1 = append(e1, fmt.Errorf("Error 1"))
	e1 = append(e1, fmt.Errorf("Error 2"))

	// Get a multierror.MultiError from it
	err := e1.Err()

	// Output: "2 errors: Error 1; Error 2"
	fmt.Println(err)

	// Iterate over the individual errors
	merr := err.(*multierror.MultiError)
	for _, err := range merr.Errors {
		fmt.Println(err) // Output: "Error 1" and "Error 2"
	}

	// If multierror.Errors contains no errors, its Err() returns nil
	var e2 multierror.Errors
	err = e2.Err()

	// Output: "<nil>"
	fmt.Println(err)
}
```
