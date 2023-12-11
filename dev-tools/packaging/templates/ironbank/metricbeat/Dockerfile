################################################################################
# Build stage 0
# Extract Metricbeat and make various file manipulations.
################################################################################
ARG BASE_REGISTRY=registry1.dsop.io
ARG BASE_IMAGE=redhat/ubi/ubi9
ARG BASE_TAG=9.3

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} as builder

ARG ELASTIC_STACK={{ beat_version }}
ARG ELASTIC_PRODUCT=metricbeat
ARG OS_AND_ARCH=linux-x86_64

RUN mkdir /usr/share/${ELASTIC_PRODUCT}
WORKDIR /usr/share/${ELASTIC_PRODUCT}
COPY --chown=1000:0 ${ELASTIC_PRODUCT}-${ELASTIC_STACK}-${OS_AND_ARCH}.tar.gz  .
RUN tar --strip-components=1 -zxf ${ELASTIC_PRODUCT}-${ELASTIC_STACK}-${OS_AND_ARCH}.tar.gz \
  && rm  ${ELASTIC_PRODUCT}-${ELASTIC_STACK}-${OS_AND_ARCH}.tar.gz
#COPY config/metricbeat.yml /usr/share/${ELASTIC_PRODUCT}

# Support arbitrary user ids
# Ensure that group permissions are the same as user permissions.
# This will help when relying on GID-0 to run Kibana, rather than UID-1000.
# OpenShift does this, for example.
# REF: https://docs.okd.io/latest/openshift_images/create-images.html
RUN chmod -R g=u /usr/share/${ELASTIC_PRODUCT}

# Create auxiliary folders and assigning default permissions.
RUN mkdir /usr/share/${ELASTIC_PRODUCT}/data /usr/share/${ELASTIC_PRODUCT}/logs && \
    chown -R root:root /usr/share/${ELASTIC_PRODUCT} && \
    find /usr/share/${ELASTIC_PRODUCT} -type d -exec chmod 0750 {} \; && \
    find /usr/share/${ELASTIC_PRODUCT} -type f -exec chmod 0640 {} \; && \
    chmod 0750 /usr/share/${ELASTIC_PRODUCT}/${ELASTIC_PRODUCT} && \
    chmod 0770 /usr/share/${ELASTIC_PRODUCT}/data /usr/share/${ELASTIC_PRODUCT}/logs

################################################################################
# Build stage 1
# Copy prepared files from the previous stage and complete the image.
################################################################################
FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

ARG ELASTIC_PRODUCT=metricbeat

COPY LICENSE /licenses/elastic-${ELASTIC_PRODUCT}

# Add a dumb init process
COPY tinit /tinit
RUN chmod +x /tinit
# Help with supporting the override in the ECK entrypoint
# https://github.com/elastic/cloud-on-k8s/blob/272fd0f2b344b1f86f04decb561eceab8a5a3254/pkg/controller/agent/pod.go#L455
# TODO: eventually /tinit will be replaced by /usr/bin/tini
RUN ln -s /tinit /usr/bin/tini

# Bring in product from the initial stage.
COPY --from=builder --chown=1000:0 /usr/share/${ELASTIC_PRODUCT} /usr/share/${ELASTIC_PRODUCT}
WORKDIR /usr/share/${ELASTIC_PRODUCT}
RUN ln -s /usr/share/${ELASTIC_PRODUCT} /opt/${ELASTIC_PRODUCT}

ENV ELASTIC_CONTAINER="true"
RUN ln -s /usr/share/${ELASTIC_PRODUCT}/${ELASTIC_PRODUCT} /usr/bin/${ELASTIC_PRODUCT}

# Support arbitrary user ids
# Ensure gid 0 write permissions for OpenShift.
RUN chmod -R g+w /usr/share/${ELASTIC_PRODUCT}

# config file ("${ELASTIC_PRODUCT}.yml") can only be writable by the root and group root
# it is needed on some configurations where the container needs to run as root
RUN chown root:root /usr/share/${ELASTIC_PRODUCT}/${ELASTIC_PRODUCT}.yml \
  && chmod go-w /usr/share/${ELASTIC_PRODUCT}/${ELASTIC_PRODUCT}.yml \
  && chmod go-w /usr/share/${ELASTIC_PRODUCT}/modules.d/system.yml

# Remove the suid bit everywhere to mitigate "Stack Clash"
RUN find / -xdev -perm -4000 -exec chmod u-s {} +

# Provide a non-root user to run the process.
RUN groupadd --gid 1000 ${ELASTIC_PRODUCT} && useradd --uid 1000 --gid 1000 --groups 0 --home-dir /usr/share/${ELASTIC_PRODUCT} --no-create-home ${ELASTIC_PRODUCT}

USER ${ELASTIC_PRODUCT}
ENV ELASTIC_PRODUCT=${ELASTIC_PRODUCT}

# TODO: eventually /tinit will be replaced by /usr/bin/tini
ENTRYPOINT ["/tinit", "--", "/usr/share/metricbeat/metricbeat", "-E", "http.enabled=true", "-E", "http.host=unix:///usr/share/metricbeat/data/metricbeat.sock"]
CMD ["-environment", "container"]

# see https://www.elastic.co/guide/en/beats/metricbeat/current/http-endpoint.html
HEALTHCHECK --interval=10s --timeout=5s --start-period=1m --retries=5 CMD curl -I -f --max-time 5 --unix-socket '/usr/share/metricbeat/data/metricbeat.sock' 'http:/stats/?pretty'
