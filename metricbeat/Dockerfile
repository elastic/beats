FROM golang:1.17.5

RUN \
    apt update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -qq -y --no-install-recommends \
         netcat \
         python3 \
         python3-dev \
         python3-pip \
         python3-venv \
         libaio-dev \
         unzip \
      && rm -rf /var/lib/apt/lists/*

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2

# Oracle instant client
RUN cd /usr/lib \
  && curl -sLo instantclient-basic-linux.zip https://download.oracle.com/otn_software/linux/instantclient/19600/instantclient-basic-linux.x64-19.6.0.0.0dbru.zip \
  && unzip instantclient-basic-linux.zip \
  && rm instantclient-basic-linux.zip
ENV LD_LIBRARY_PATH=/usr/lib/instantclient_19_6

ENV PATH=/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/lib/oracle/12.2/client64/bin

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
