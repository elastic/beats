# build a Python virtualenv
FROM golang:1.19.10 as virtualenv
RUN \
    apt update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -qq -y --no-install-recommends \
        build-essential \
        libssl-dev \
        zlib1g-dev \
        libbz2-dev \
        libreadline-dev \
        libsqlite3-dev \
        wget \
        curl \
        llvm \
        libncurses5-dev \
        libncursesw5-dev \
        xz-utils \
        tk-dev \
        libffi-dev \
        liblzma-dev \
        && rm -rf /var/lib/apt/lists/*

# The project requires Python 3.9, but Debian 12 only has 3.10. We can't use the official Python
# see .python-version file in the root of the project for the version we need.
# https://github.com/pyenv/pyenv-installer
# https://github.com/pyenv/pyenv
ENV PYENV_GIT_TAG=v2.3.22
ENV PYTHON_VERSION=3.9.13
RUN curl -sS https://pyenv.run | bash
ENV PYENV_ROOT="/root/.pyenv"
ENV PATH="${PYENV_ROOT}/bin:${PYENV_ROOT}/versions/${PYTHON_VERSION}/bin:${PATH}"
RUN pyenv install ${PYTHON_VERSION}
RUN pyenv global ${PYTHON_VERSION}
RUN [ "$(python3 --version)" = "Python 3.9.13" ]
RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade docker-compose==1.23.2
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade PyYAML==5.4.1

# Build the final image
FROM golang:1.19.10
RUN \
    apt update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -qq -y --no-install-recommends \
         netcat-openbsd \
         libaio-dev \
         unzip \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.19.10 uses Debian 12 which now enforces PEP668.
ENV PYTHON_VERSION=3.9.13
ENV PYTHON_HOME=/root/.pyenv/versions/${PYTHON_VERSION}
COPY --from=virtualenv ${PYTHON_HOME} ${PYTHON_HOME}
ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${PYTHON_HOME}/lib
ENV PATH="${PYTHON_HOME}/bin:${PATH}"
# create a virtualenv
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv ${VIRTUAL_ENV}
ENV PATH="${VIRTUAL_ENV}/bin:${PATH}"
# test the Python version is the one we want
RUN [ "$(python3 --version)" = "Python ${PYTHON_VERSION}" ]
# # Oracle instant client
RUN cd /usr/lib \
  && curl -sLo instantclient-basic-linux.zip https://download.oracle.com/otn_software/linux/instantclient/19600/instantclient-basic-linux.x64-19.6.0.0.0dbru.zip \
  && unzip instantclient-basic-linux.zip \
  && rm instantclient-basic-linux.zip
ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:/usr/lib/instantclient_19_6

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
