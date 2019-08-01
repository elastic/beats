ARG GO_VERSION=1.12.7
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


RUN go build -o x-pack/pkg/core/plugin/operation/tests/scripts/configurable-1.0-darwin-x86/configurable x-pack/pkg/core/plugin/operation/tests/scripts/configurable-1.0-darwin-x86/main.go
RUN go build -o x-pack/pkg/core/plugin/operation/tests/scripts/configurablebyfile-1.0-darwin-x86/configurablebyfile x-pack/pkg/core/plugin/operation/tests/scripts/configurablebyfile-1.0-darwin-x86/main.go
