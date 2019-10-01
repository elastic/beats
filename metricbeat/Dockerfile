FROM golang:1.12.9

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

RUN mkdir -p /opt/mqm \
  && chmod a+rx /opt/mqm

# Location of the downloadable MQ client package \
ENV RDURL="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist" \
    RDTAR="IBM-MQC-Redist-LinuxX64.tar.gz" \
    VRMF=9.1.3.0

# Install the MQ client from the Redistributable package. This also contains the
# header files we need to compile against.
RUN cd /opt/mqm \
 && curl -LO "$RDURL/$VRMF-$RDTAR" \
 && tar -zxf ./*.tar.gz \
 && rm -f ./*.tar.gz

ENV CGO_CFLAGS="-I/opt/mqm/inc/"
ENV CGO_LDFLAGS="-L$/opt/mqm/lib64 -Wl,-rpath,/opt/mqm/lib64"
ENV CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
