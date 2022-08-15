<<<<<<< HEAD
FROM golang:1.17.10
=======
FROM golang:1.18.5
>>>>>>> 56ca6f5c4d ([automation] Update go release version 1.18.5 (#32574))

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         python3 \
         python3-pip \
         python3-venv \
         librpm-dev \
         iproute2 \
      && rm -rf /var/lib/apt/lists/*

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2
