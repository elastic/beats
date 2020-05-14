FROM debian:stretch

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
        google-cloud-sdk \
        google-cloud-sdk-pubsub-emulator \
	&& rm -rf /var/lib/apt/lists/*

RUN \
    mkdir /data

HEALTHCHECK --interval=1s --retries=90 CMD curl -s -f http://localhost:8432/

CMD gcloud beta emulators pubsub start --data-dir /data --host-port "0.0.0.0:8432"
