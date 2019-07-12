FROM store/oracle/database-instantclient:12.2.0.1 as oracle
FROM golang:1.12.4

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         netcat \
         python-pip \
         virtualenv \
      && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip
RUN pip install --upgrade setuptools
RUN pip install --upgrade docker-compose==1.23.2

# Oracle instant client
COPY --from=oracle /usr/lib/oracle/12.2 /usr/lib/oracle/12.2
COPY --from=oracle /usr/lib64 /usr/lib64
ENV LD_LIBRARY_PATH=/usr/lib/oracle/12.2/client64/lib:/usr/lib64
ENV PATH=/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/lib/oracle/12.2/client64/bin

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
