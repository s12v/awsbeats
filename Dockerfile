# See https://stackoverflow.com/a/48324849 for how ARG before FROM works
# Used from within the first FROM
ARG GO_VERSION=${GO_VERSION:-1.10.2}
# Used from within the second FROM
ARG BEAT_DOCKER_IMAGE

FROM golang:${GO_VERSION} AS awsbeats

LABEL maintainr "Yusuke KUOKA <ykuoka@gmail.com>"

# "MAINTAINER" is deprecated in favor of the above label. But required to make codacy happy
MAINTAINER Yusuke KUOKA <ykuoka@gmail.com>

COPY . /go/src/github.com/s12v/awsbeats

WORKDIR /go/src/github.com/s12v/awsbeats

ARG BEATS_VERSION=${BEATS_VERSION:-6.1.2}
ARG GO_PLATFORM=${GO_PLATFORM:-linux-amd64}
ARG AWSBEATS_VERSION=${AWSBEATS_VERSION:-1-snapshot}
ARG BEAT_NAME=${BEAT_NAME:-filebeat}
RUN curl --verbose --fail https://raw.githubusercontent.com/golang/dep/master/install.sh -o install.sh && sh install.sh && rm install.sh
RUN go get github.com/elastic/beats || true
RUN /go/bin/dep ensure
# You need to enable CGO on both the plugin and the beat.
# Otherwise, for example, filebeat w/ CGO fails to load the plugin w/o CGO, emitting an error like:
#   Exiting: plugin.Open("kinesis"): plugin was built with a different version of package net
RUN CGO_ENABLED=1 GOOS=linux make build
FROM golang:${GO_VERSION} AS beats

LABEL maintainr "Yusuke KUOKA <ykuoka@gmail.com>"

COPY Makefile /build/Makefile

WORKDIR /build

ARG BEATS_VERSION=${BEATS_VERSION:-6.1.2}
ARG GO_VERSION=${GO_VERSION:-1.10.2}
ARG GO_PLATFORM=${GO_PLATFORM:-linux-amd64}
ARG BEAT_NAME=${BEAT_NAME:-filebeat}
ARG BEAT_GITHUB_REPO
ARG BEAT_GO_PKG

#RUN go get github.com/elastic/beats || true
# Beats requires CGO for plugin support as per https://github.com/elastic/beats/commit/d21decb720e7fdeb986f4ebac413cc816353aa55
RUN CGO_ENABLED=1 make beats && \
  pwd && find ./target

FROM ${BEAT_DOCKER_IMAGE}

LABEL maintainr "Yusuke KUOKA <ykuoka@gmail.com>"

ARG AWSBEATS_VERSION=${AWSBEATS_VERSION:-1-snapshot}
ARG BEATS_VERSION=${BEATS_VERSION:-6.1.2}
ARG GO_VERSION=${GO_VERSION:-1.10.2}
ARG GO_PLATFORM=${GO_PLATFORM:-linux-amd64}
ARG BEAT_NAME=${BEAT_NAME:-filebeat}

COPY --from=awsbeats /go/src/github.com/s12v/awsbeats/target/kinesis-${AWSBEATS_VERSION}-${BEATS_VERSION}-go${GO_VERSION}-linux-amd64.so /usr/share/${BEAT_NAME}/kinesis.so
COPY --from=beats /build/target/${BEAT_NAME}-${BEATS_VERSION}-go${GO_VERSION}-linux-amd64 /usr/share/${BEAT_NAME}/${BEAT_NAME}

# Usage:
#   docker run --rm s12v/awsbeats:canary cat filebeat.yml > filebeat.yml
#   cat outputs.yml >> filebeat.yml
#   docker run --rm -v $(pwd)/filebeat.yml:/etc/filebeat/filebeat.yml s12v/awsbeats:canary filebeat --plugin kinesis.so -e -v 
