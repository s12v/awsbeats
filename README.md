[![Build Status](https://travis-ci.org/s12v/awsbeats.svg?branch=master)](https://travis-ci.org/s12v/awsbeats)
[![codecov](https://codecov.io/gh/s12v/awsbeats/branch/master/graph/badge.svg)](https://codecov.io/gh/s12v/awsbeats)

# AWS Beats

Experimental [Filebeats](https://github.com/elastic/beats/filebeats) output plugin.
Supports AWS Kinesis Firehose streams.

## Build

Build requires go 1.10
```
make
```
or `go build -buildmode=plugin ./plugins/firehose`

## Run

```
./filebeat -e -plugin firehose.so -d '*'
```

TODO: mention that filebeat has to be built using the same Go version

## Configuration

Add to `filebeats.yml`:
```
output.firehose:
  region: eu-central-1
  stream_name: test1
```
