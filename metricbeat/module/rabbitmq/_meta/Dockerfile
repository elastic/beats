FROM rabbitmq:3.7.4-management

RUN apt-get update && apt-get install -y netcat && apt-get clean
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 -v 127.0.0.1 15672 </dev/null
EXPOSE 15672
