# awsbeats
Experimental filebeats plugin


## Build

Build requires go 1.10
```
go build -buildmode=plugin ./plugins/firehose
./filebeat -e -plugin firehose.so -d '*'
```

## Configuration

Add to `filebeats.yml`:
```
output.firehose:
    region: eu-central-1
    stream_name: test1
```
