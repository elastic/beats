ARG MSSQL_VERSION
FROM mcr.microsoft.com/mssql/server:${MSSQL_VERSION}

ENV ACCEPT_EULA='Y'
ENV SA_PASSWORD='1234_asdf'

# Use the same healthcheck as the Windows version of the image.
# https://github.com/Microsoft/mssql-docker/blob/a3020afeec9be1eb2d67645ac739438eb8f2c545/windows/mssql-server-windows/dockerfile#L31
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD /opt/mssql-tools18/bin/sqlcmd -S localhost -U SA -P ${SA_PASSWORD} -No -Q "SELECT 1" || exit 1
