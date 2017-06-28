FROM aerospike:3.9.0

RUN apt-get update && apt-get install -y netcat
HEALTHCHECK CMD nc -z localhost 3000
