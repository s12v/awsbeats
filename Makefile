.PHONY: all

all: test build

test:
		go test ./firehose -v -coverprofile=coverage.txt -covermode=atomic

build:
		go build -buildmode=plugin ./plugins/firehose
