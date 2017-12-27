.PHONY: all

GO_VERSION=$(shell go version | cut -d ' ' -f 3,4 | sed -e 's/ /-/g' | sed -e 's/\//-/g')

all: test build

test:
	go test ./firehose -v -coverprofile=coverage.txt -covermode=atomic

build:
	go build -buildmode=plugin ./plugins/firehose
	@mkdir -p "$(CURDIR)/target"
ifdef BEATS_VERSION
	@echo "Building filebeats $(BEATS_VERSION)..."

	@cd "$$GOPATH/src/github.com/elastic/beats/filebeat" &&\
	git checkout $(BEATS_VERSION) &&\
	make &&\
	mv filebeat "$(CURDIR)/target/filebeat-$(BEATS_VERSION)-$(GO_VERSION)"

ifdef AWSBEATS_VERSION
	@mv firehose.so "$(CURDIR)/target/firehose.so-$(AWSBEATS_VERSION)-$(BEATS_VERSION)-$(GO_VERSION)"
else
	@mv firehose.so "$(CURDIR)/target/firehose.so-$(BEATS_VERSION)-$(GO_VERSION)"
endif

else
	@mv firehose.so "$(CURDIR)/target/firehose.so-$(GO_VERSION)"
endif
	@ls -l "$(CURDIR)/target"
