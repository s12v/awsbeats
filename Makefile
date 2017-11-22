.PHONY: all

all: test build

test:
		go test ./firehose -v

build:
		go build -buildmode=plugin ./plugins/firehose
