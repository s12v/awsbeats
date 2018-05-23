.PHONY: all

GO_VERSION=$(shell go version | cut -d ' ' -f 3 | sed -e 's/ /-/g' | sed -e 's/\//-/g' | sed -e 's/^go//g')
GO_PLATFORM ?= $(go version | cut -d ' ' -f 4 | sed -e 's/ /-/g' | sed -e 's/\//-/g')
BEATS_VERSION ?= "master"
BEATS_TAG ?= $(shell echo ${BEATS_VERSION} | sed 's/[^[:digit:]]*\([[:digit:]]*\(\.[[:digit:]]*\)\)/v\1/')
AWSBEATS_VERSION ?= "1-snapshot"
DOCKER_IMAGE ?= s12v/awsbeats
DOCKER_TAG ?= canary

all: test beats build

test:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)"
	go test ./firehose -v -coverprofile=coverage.txt -covermode=atomic
	go test ./streams -v -coverprofile=coverage.txt -covermode=atomic

format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)" || \
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -w {} + | tee /dev/stderr)"

build: format
	@echo "Building the plugin with: BEATS_VERSION=$(BEATS_VERSION) GO_VERSION=$(GO_VERSION) GO_PLATFORM=$(GO_PLATFORM)"
	@cd "$$GOPATH/src/github.com/elastic/beats/filebeat" && \
	git checkout $(BEATS_TAG) && \
	cd "$(CURDIR)"
	go build -buildmode=plugin ./plugins/kinesis
	@mkdir -p "$(CURDIR)/target"
	@mv kinesis.so "$(CURDIR)/target/kinesis-$(AWSBEATS_VERSION)-$(BEATS_VERSION)-go$(GO_VERSION)-$(GO_PLATFORM).so"
	@find "$(CURDIR)"/target/

beats:
ifdef BEATS_VERSION
	@echo "Building filebeats:$(BEATS_VERSION)..."
	@mkdir -p "$(CURDIR)/target"
	@cd "$$GOPATH/src/github.com/elastic/beats/filebeat" &&\
	git checkout $(BEATS_TAG) &&\
	make &&\
	mv filebeat "$(CURDIR)/target/filebeat-$(BEATS_VERSION)-go$(GO_VERSION)-$(GO_PLATFORM)"
else
	$(error BEATS_VERSION is undefined)
endif

.PHONY: dockerimage
dockerimage:
	docker build --build-arg AWSBEATS_VERSION=$(AWSBEATS_VERSION) --build-arg GO_VERSION=$(GO_VERSION) --build-arg BEATS_VERSION=$(BEATS_VERSION) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
