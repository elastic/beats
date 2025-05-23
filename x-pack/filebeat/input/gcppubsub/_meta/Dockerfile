FROM debian:stretch
ARG SDK_VERSION

# TODO: use newer base
RUN sed -i s/deb.debian.org/archive.debian.org/g /etc/apt/sources.list \
    && sed -i 's|security.debian.org|archive.debian.org/|g' /etc/apt/sources.list \
    && sed -i '/stretch-updates/d' /etc/apt/sources.list 

RUN \
    apt-get update \
	&& apt-get install -y \
	    apt-transport-https \
	    ca-certificates \
	    curl \
		python \
		openjdk-8-jre \
		gnupg2 \
	&& rm -rf /var/lib/apt/lists/*

RUN \
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
        >> /etc/apt/sources.list.d/google-cloud-sdk.list \
    && curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg \
        | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -

RUN \
    apt-get update \
    && apt-get install -y \
        google-cloud-sdk=${SDK_VERSION} \
        google-cloud-sdk-pubsub-emulator=${SDK_VERSION} \
	&& rm -rf /var/lib/apt/lists/*

RUN \
    mkdir /data

HEALTHCHECK --interval=1s --retries=90 CMD curl -s -f --http2 http://localhost:8432/

CMD gcloud beta emulators pubsub start --data-dir /data --host-port "0.0.0.0:8432"
