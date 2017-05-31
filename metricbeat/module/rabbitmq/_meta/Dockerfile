FROM rabbitmq:3-management

RUN apt-get update && apt-get install -y netcat && apt-get clean
HEALTHCHECK CMD nc -w 1 -v 127.0.0.1 15672 </dev/null
EXPOSE 15672
