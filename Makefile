.PHONY: all

GO_VERSION=$(shell go version | cut -d ' ' -f 3,4 | sed -e 's/ /-/g' | sed -e 's/\//-/g')
BEATS_VERSION ?= "master"
AWSBEATS_VERSION ?= "1-snapshot"

all: test beats build

test:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)"
	go test ./firehose -v -coverprofile=coverage.txt -covermode=atomic
	go test ./streams -v -coverprofile=coverage.txt -covermode=atomic

format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)" || \
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -w {} + | tee /dev/stderr)"

build: format
	go build -buildmode=plugin ./plugins/kinesis
	@mkdir -p "$(CURDIR)/target"
	@mv kinesis.so "$(CURDIR)/target/kinesis-$(AWSBEATS_VERSION)-$(BEATS_VERSION)-$(GO_VERSION).so"

beats:
ifdef BEATS_VERSION
	@echo "Building filebeats:$(BEATS_VERSION)..."
	@mkdir -p "$(CURDIR)/target"
	@cd "$$GOPATH/src/github.com/elastic/beats/filebeat" &&\
	git checkout $(BEATS_VERSION) &&\
	make &&\
	mv filebeat "$(CURDIR)/target/filebeat-$(BEATS_VERSION)-$(GO_VERSION)"
else
	$(error BEATS_VERSION is undefined)
endif
