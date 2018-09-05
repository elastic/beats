FROM microsoft/mssql-server-linux:2017-latest
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 1433

ENV ACCEPT_EULA='Y'
ENV SA_PASSWORD='1234_asdf'
