[![Build Status](https://travis-ci.org/s12v/awsbeats.svg?branch=master)](https://travis-ci.org/s12v/awsbeats)
[![codecov](https://codecov.io/gh/s12v/awsbeats/branch/master/graph/badge.svg)](https://codecov.io/gh/s12v/awsbeats)

# AWS Beats

Experimental [Beat](https://github.com/elastic/beats) output plugin.
Tested with Filebeat and Metricbeat. Supports AWS Kinesis Data Streams and Data Firehose.

__NOTE: Beat and the plugin should be built using the same Golang version.__

## Quick start

Either:

- Download binary files from https://github.com/s12v/awsbeats/releases
- Pull docker images from [`kubeaws/awsbeats`](https://hub.docker.com/r/kubeaws/awsbeats/). Note that the docker repository is subject to change in near future.

### Firehose

- Add to `filebeats.yml`:
```
output.firehose:
  region: eu-central-1
  stream_name: test1 # Your delivery stream name
```
- Run filebeat with plugin `./filebeat-v6.1.3-go1.10rc1-linux-amd64 -plugin kinesis.so-0.0.3-v6.1.3-go1.10rc1-linux-amd64`

### Streams

- Download binary files from https://github.com/s12v/awsbeats/releases
- Add to `filebeats.yml`:
```
output.streams:
  region: eu-central-1
  stream_name: test1 # Your stream name
  partition_key: mykey # In case your beat event is {"foo":1,"mykey":"bar"}, not "mykey" but "bar" is used as the partition key
```
See the example [filebeat.yaml](https://github.com/s12v/awsbeats/blob/master/example/streams/filebeat.yml) for more details.

- Run filebeat with plugin `./filebeat-v6.1.3-go1.10rc1-linux-amd64 -plugin kinesis.so-0.0.3-v6.1.3-go1.10rc1-linux-amd64`

## AWS authentication

- Default AWS credentials chain is used (environment, credentials file, EC2 role)
- Assume role is not supported 

## Build it yourself

Build requires Go 1.10+ and Linux. You need to define Filebeat version (`v6.1.3` in this example)

```
go get github.com/elastic/beats
# curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
dep ensure
make BEATS_VERSION=v6.1.3
```

In `target/` you will find filebeat and plugin, for example:
```
filebeat-v6.1.3-go1.10rc1-linux-amd64
kinesis.so-1-snapshot-v6.1.3-go1.10rc1-linux-amd64
```

## Running in a docker container

To build a docker image for awsbeats, run `make dockerimage`.

### filebeat

```
make dockerimage BEATS_VERSION=6.2.4 GO_VERSION=1.10.2 GOPATH=$HOME/go
```

There is also a convenient make target `filebeat-image` with sane defaults:

```console
make filebeat-image
```

The resulting docker image is tagged `s12v/awsbeats:filebeat-canary`.  It contains a custom build of filebeat and the plugin, along with all the relevant files from the official filebeat docker image.

To try running it, provide AWS credentials via e.g. envvars and run `hack/dockerized-filebeat`:

```
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

hack/containerized-filebeat
```

Emit some line-delimited json log messages:

```
hack/emit-ndjson-logs
```

### metricbeat

**metricbeat**:

```
make dockerimage BEATS_VERSION=6.2.4 GO_VERSION=1.10.2 BEAT_NAME=metricbeat GOPATH=$HOME/go

# Or:

make metricbeat-image
```

### apm-server

```
make apm-server-image

hack/containerized-apm-server
```

### auditbeat

```
make auditbeat-image

hack/containerized-auditbeat
```

### heartbeat

```
make heartbeat-image

hack/containerized-heartbeat
```

## Running awsbeats on a Kubernetes cluster

### Filebeat

Use the helm chart:

```
cat << EOS > values.yaml
image:
  repository: kubeaws/awsbeats
  tag: canary
  pullPolicy: Always

plugins:
  - kinesis.so

config:
  output.file:
    enabled: false
  output.streams:
    enabled: true
    region: ap-northeast-1
    stream_name: test1
    partition_key: mykey
EOS

# No need to do this once stable/filebeat v0.3.0 is published
# See https://github.com/kubernetes/charts/pull/5698
git clone git@github.com:kubernetes/charts.git charts

helm upgrade --install filebeat ./charts/stable/filebeat \
  -f values.yaml \
  --set rbac.enabled=true
```

### APM Server

```
cat << EOS > values.yaml
image:
  repository: kubeaws/awsbeats
  tag: apm-server-canary
  pullPolicy: Always

plugins:
  - kinesis.so

config:
  output.file:
    enabled: false
  output.streams:
    enabled: true
    region: ap-northeast-1
    stream_name: test1
    partition_key: mykey
EOS

# No need to do this once stable/apm-server is merged
# See https://github.com/kubernetes/charts/pull/6058
git clone git@github.com:mumoshu/charts.git charts
git checkout apm-server

helm upgrade --install apm-server ./charts/stable/apm-server \
  -f values.yaml \
  --set rbac.enabled=true
```

### Metricbeat

Edit the official Kubernetes manifests to use:

- custom metricbeat docker image
- `streams` output instead of the default `elasticsearch` one

```
( find ~/go/src/github.com/elastic/beats/deploy/kubernetes/metricbeat -name '*.yaml' -not -name metricbeat-daemonset-configmap.yaml -exec bash -c 'sed -e '"'"'s/image: .*/image: "kubeaws\/awsbeats:metricbeat-canary"/g'"'"' {} | sed -e '"'"'s/"-e",/"-e", "-plugin", "kinesis.so",/g'"'" \; -exec echo '---' \; ) > metricbeat.all.yaml

kubectl create -f example/metricbeat/metricbeat.configmap.yaml -f metricbeat.all.yaml
```

### Trouble-shooting

If you see `No such file or directory` error of filebeat while building the plugin, you likely to be relying on the default `GOPATH`. It is `$HOME/go` in recent versions of golang.

If you got affected by this, try running:

```
$ make BEATS_VERSION=v6.1.3 GOPATH=$HOME/go
```

## Output buffering

TODO
