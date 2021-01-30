# Test Vue.js Examples

This suite tests the examples that ship with the open source Vue.js project.

You can run the test suites in two ways

## Running via `@elastic/synthetics`

We can invoke the Synthetics runner from the CLI using the below steps

```sh
// Install the dependencies
npm install

// Invoke the runner and show test results
npx @elastic/synthetics .

```

## Running via `Heartbeat`

Invoke the synthetic test suites using heartbeat.

```sh
// Run the below command inside /examples/docker directory

// run heartbeat which is already configured to run the todo app, you
// can check `heartbeat.docker.yml`
./run-build-local.sh -E output.console={}
```
