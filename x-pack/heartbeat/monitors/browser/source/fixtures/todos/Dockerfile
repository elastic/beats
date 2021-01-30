# This Dockerfile illustrates the usage of synthetics in an air gapped environment, where 
# public internet access is not available. In this situation you'll want to create
# a custom image using our official docker image as a base.

# Use our synthetics image as a base.
ARG STACK_VERSION=latest
FROM docker.elastic.co/experimental/synthetics:${STACK_VERSION}-synthetics
# Use the line below if you're using a custom base image, usually only for
# developers
#FROM heartbeat-synthetics-local

# This flag variable will prevent heartbeat from running `npm i` or
# similar commands that depend on an internet connection.
# We'll have to do that work now, when we bake the image.
ENV ELASTIC_SYNTHETICS_OFFLINE=true

# Copy our heartbeat config directly to the image.
# This could be done as a shared mount instead, but if we're
# baking an image anyway, this may be something that may be easier
# to do in this manner.
COPY heartbeat.docker.yml /usr/share/heartbeat/heartbeat.yml

RUN mkdir -p $SUITES_DIR/todos
# Copy your custom synthetics tests into a folder on the image
COPY . $SUITES_DIR/todos/

# Install NPM deps locally on this image
# Please note that it's important to run both `npm install` AND `npm install playwright`
# for more see this issue: https://github.com/microsoft/playwright/issues/3712
RUN cd $SUITES_DIR/todos && npm install
