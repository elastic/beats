ARG GO_VERSION=1.17.6
FROM circleci/golang:${GO_VERSION}


ARG TEST_RESULTS=/tmp/test-results

RUN mkdir -p ${TEST_RESULTS} && mkdir -p ./code
RUN go get github.com/magefile/mage

ENV GO111MODULE=on
WORKDIR ./code
#COPY --chown=circleci:circleci . .
COPY . .
VOLUME "/tmp" "dev-tools/mage/build/distributions"
USER root

