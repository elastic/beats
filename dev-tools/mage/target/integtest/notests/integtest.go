package notests

import "fmt"

// IntegTest method fallbacks to GoIntegTest()
func IntegTest() {
	GoIntegTest()
}

// GoIntegTest method informs that no integration tests will be executed.
func GoIntegTest() {
	fmt.Println(">> integTest: Complete (no tests require the integ test environment)")
}
