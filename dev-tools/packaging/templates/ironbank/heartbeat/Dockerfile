################################################################################
# Build stage 0
# Extract Heartbeat and make various file manipulations.
################################################################################
ARG BASE_REGISTRY=registry1.dsop.io
ARG BASE_IMAGE=ironbank/redhat/ubi/ubi8
ARG BASE_TAG=8.4

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} as prep_files

ARG ELASTIC_STACK={{ beat_version }}
ARG ELASTIC_PRODUCT=heartbeat
ARG OS_AND_ARCH=linux-x86_64

RUN mkdir /usr/share/${ELASTIC_PRODUCT}
WORKDIR /usr/share/${ELASTIC_PRODUCT}
COPY --chown=1000:0 ${ELASTIC_PRODUCT}-${ELASTIC_STACK}-${OS_AND_ARCH}.tar.gz  .
RUN tar --strip-components=1 -zxf ${ELASTIC_PRODUCT}-${ELASTIC_STACK}-${OS_AND_ARCH}.tar.gz \
  && rm  ${ELASTIC_PRODUCT}-${ELASTIC_STACK}-${OS_AND_ARCH}.tar.gz
#COPY config/heartbeat.yml /usr/share/${ELASTIC_PRODUCT}

# Support arbitrary user ids
# Ensure that group permissions are the same as user permissions.
# This will help when relying on GID-0 to run Kibana, rather than UID-1000.
# OpenShift does this, for example.
# REF: https://docs.okd.io/latest/openshift_images/create-images.html
RUN chmod -R g=u /usr/share/${ELASTIC_PRODUCT}

# Create auxiliar folders and assing default permissions.
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

ARG ELASTIC_PRODUCT=heartbeat

COPY LICENSE /licenses/elastic-${ELASTIC_PRODUCT}

# Install synthetics dependencies.
COPY signatures/* *.rpm /
RUN rpm --import linux_signing_key.pub RPM-GPG-KEY-CentOS-Official RPM-GPG-KEY-rockyofficial \
  && yum -y --allowerasing --nodocs --nobest --setopt=install_weak_deps=False install *.rpm \
  && rm -r *.rpm RPM-GPG-KEY-CentOS-Official linux_signing_key.pub RPM-GPG-KEY-rockyofficial \
  && yum clean all && rm -rf /var/cache/yum

ENV NODE_PATH=/usr/share/${ELASTIC_PRODUCT}/.node
RUN echo \
    $NODE_PATH \
    /usr/share/${ELASTIC_PRODUCT}/.config \
    /usr/share/${ELASTIC_PRODUCT}/.synthetics \
    /usr/share/${ELASTIC_PRODUCT}/.npm \
    /usr/share/${ELASTIC_PRODUCT}/.cache \
    | xargs -IDIR sh -c 'mkdir -p DIR && chmod 0770 DIR'

# Add a dumb init process
COPY tinit /tinit
RUN chmod +x /tinit

# Bring in product from the initial stage.
COPY --from=prep_files --chown=1000:0 /usr/share/${ELASTIC_PRODUCT} /usr/share/${ELASTIC_PRODUCT}
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
  && chmod go-w /usr/share/${ELASTIC_PRODUCT}/${ELASTIC_PRODUCT}.yml

# Remove the suid bit everywhere to mitigate "Stack Clash"
RUN find / -xdev -perm -4000 -exec chmod u-s {} +

# Provide a non-root user to run the process.
RUN groupadd --gid 1000 ${ELASTIC_PRODUCT} && useradd --uid 1000 --gid 1000 --groups 0 --home-dir /usr/share/${ELASTIC_PRODUCT} --no-create-home ${ELASTIC_PRODUCT}

RUN chown -R ${ELASTIC_PRODUCT} $NODE_PATH

USER ${ELASTIC_PRODUCT}
ENV ELASTIC_PRODUCT=${ELASTIC_PRODUCT}

# Setup synthetics env vars
ENV ELASTIC_SYNTHETICS_CAPABLE=true
ENV SUITES_DIR=/usr/share/${ELASTIC_PRODUCT}/suites
ENV PATH="$NODE_PATH/node/bin:$PATH"
# Install the latest version of @elastic/synthetics forcefully ignoring the previously
# cached node_modules, heartbeat then calls the global executable to run test suites
# Setup node
COPY node.tar.gz /usr/share/heartbeat/.node
COPY elastic-synthetics.tar.gz /usr/share/heartbeat/.node
RUN cd /usr/share/heartbeat/.node \
  && mkdir -p node \
  && tar -xzf "node.tar.gz" --strip 1 -C node \
  && rm "node.tar.gz" \
  && npm i -g --offline -f "elastic-synthetics.tar.gz" \
  && rm "elastic-synthetics.tar.gz" \
  && chmod ug+rwX -R $NODE_PATH

ENTRYPOINT ["/tinit", "--", "/usr/share/heartbeat/heartbeat", "-E", "http.enabled=true", "-E", "http.host=unix:///usr/share/heartbeat/data/heartbeat.sock"]
CMD ["-environment", "container"]

# see https://www.elastic.co/guide/en/beats/heartbeat/current/http-endpoint.html
HEALTHCHECK --interval=10s --timeout=5s --start-period=1m --retries=5 CMD curl -I -f --max-time 5 --unix-socket '/usr/share/heartbeat/data/heartbeat.sock' 'http:/stats/?pretty'
