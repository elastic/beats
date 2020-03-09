Fakewebapi is a simple test only Webserver

The server implements the minimal calls and response to do high level testing of the agent:

- Enroll successfully an Agent.
- Allow an Agent to periodically check in.


By default the server will return an empty list of actions, it's possible at runtime to change the returned
data by using the `push.sh` script. The script will POST a JSON document to return on the next request.

Read the code of `push.sh` and the `fetch.sh` script for the usage information.

