# Guidelines

This document contains architecture details around Elastic Agent and guidelines on how new inputs and processes should be built.

## Processes running as service and error handling

All the processes started by Elastic Agent are running as service. Each service is expected to handle local errors on its own and continue working. A process should only fail on startup if an invalid configuration is passed in. As soon as a process is running and partial updates to the config are made without restart, the service is expected to keep running but report the errors.

A service that needs to do setup tasks on startup is expected to retry until it succeeds and not error out after a certain timeout.
