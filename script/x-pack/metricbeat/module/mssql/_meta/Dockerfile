ARG MSSQL_VERSION
FROM microsoft/mssql-server-linux:${MSSQL_VERSION}

ENV ACCEPT_EULA='Y'
ENV SA_PASSWORD='1234_asdf'

# Use the same healthcheck as the Windows version of the image.
# https://github.com/Microsoft/mssql-docker/blob/a3020afeec9be1eb2d67645ac739438eb8f2c545/windows/mssql-server-windows/dockerfile#L31
HEALTHCHECK --interval=1s --retries=90 CMD /opt/mssql-tools/bin/sqlcmd -U SA -P ${SA_PASSWORD} -Q "select 1"
