[![Build Status](https://travis-ci.org/s12v/awsbeats.svg?branch=master)](https://travis-ci.org/s12v/awsbeats)
[![codecov](https://codecov.io/gh/s12v/awsbeats/branch/master/graph/badge.svg)](https://codecov.io/gh/s12v/awsbeats)

# AWS Beats

Experimental [Filebeat](https://github.com/elastic/beats) output plugin.
Supports AWS Kinesis Data Firehose streams.

__NOTE: Filebeat and plugin should be built using the same Golang version.__

## Quick start

- Download binary files from https://github.com/s12v/awsbeats/releases
- Add to `filebeats.yml`:
```
output.firehose:
  region: eu-central-1
  stream_name: test1 # Your delivery stream name
```
- Run it with `./filebeat -plugin firehose.so`

## AWS authentication

- Default AWS credentials chain is used (environment, credentials file, EC2 role)
- Assume role is not supported 

## Build it yourself

Build requires Go 1.10+ and Linux. You need to define Filebeat version (`v6.1.3` in this example)

```
go get github.com/elastic/beats
# curl https://glide.sh/get | sh
glide install
make BEATS_VERSION=v6.1.3
```

In `target/` you will find filebeat and plugin, for example:
```
filebeat-v6.1.3-go1.10rc1-linux-amd64*
firehose.so-1-snapshot-v6.1.3-go1.10rc1-linux-amd64
```

## Output buffering

TODO
