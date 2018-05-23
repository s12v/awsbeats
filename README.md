[![Build Status](https://travis-ci.org/s12v/awsbeats.svg?branch=master)](https://travis-ci.org/s12v/awsbeats)
[![codecov](https://codecov.io/gh/s12v/awsbeats/branch/master/graph/badge.svg)](https://codecov.io/gh/s12v/awsbeats)

# AWS Beats

Experimental [Filebeat](https://github.com/elastic/beats) output plugin.
Supports AWS Kinesis Data Firehose streams.

__NOTE: Filebeat and plugin should be built using the same Golang version.__

## Quick start

- Download binary files from https://github.com/s12v/awsbeats/releases

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
```
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

To build a docker image for awsbeats, run:

```
make dockerimage BEATS_VERSION=6.2.4 GO_VERSION=1.10.2 GOPATH=$HOME/go
```

The resulting docker image is tagged `s12v/awsbeats:canary`.  It contains a custom build of filebeat and the plugin, along with all the relevant files from the official filebeat docker image.

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

### Trouble-shooting

If you see `No such file or directory` error of filebeat while building the plugin, you likely to be relying on the default `GOPATH`. It is `$HOME/go` in recent versions of golang.

If you got affected by this, try running:

```
$ make BEATS_VERSION=v6.1.3 GOPATH=$HOME/go
```

## Output buffering

TODO
