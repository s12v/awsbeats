# awsbeats
Experimental filebeats plugin


## Build
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
