MAKE_VARIABLES := $(.VARIABLES)

GO_VERSION=$(shell go version | cut -d ' ' -f 3 | sed -e 's/ /-/g' | sed -e 's/\//-/g' | sed -e 's/^go//g')
GO_PLATFORM ?= $(shell go version | cut -d ' ' -f 4 | sed -e 's/ /-/g' | sed -e 's/\//-/g')
BEATS_VERSION ?= "master"
BEATS_TAG ?= $(shell echo ${BEATS_VERSION} | sed 's/[^[:digit:]]*\([[:digit:]]*\(\.[[:digit:]]*\)\)/v\1/')
AWSBEATS_VERSION ?= "1-snapshot"
BEAT_NAME ?= "filebeat"
DOCKER_IMAGE ?= s12v/awsbeats
DOCKER_TAG ?= $(BEAT_NAME)-canary

.PHONY: all
all: vars test beats build

.PHONY: vars
vars:
	$(info make variables for this build:)
	$(foreach v, $(filter-out $(MAKE_VARIABLES) MAKE_VARIABLES,$(.VARIABLES)), $(info $(v) = $($(v))))
	$(info To override, run make like '`make SOME_VAR=some_value`'.)

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
	@echo "Building $(BEAT_NAME):$(BEATS_VERSION)..."
	@mkdir -p "$(CURDIR)/target"
	@cd "$$GOPATH/src/github.com/elastic/beats/$(BEAT_NAME)" &&\
	git checkout $(BEATS_TAG) &&\
	make &&\
	mv $(BEAT_NAME) "$(CURDIR)/target/$(BEAT_NAME)-$(BEATS_VERSION)-go$(GO_VERSION)-$(GO_PLATFORM)"
else
	$(error BEATS_VERSION is undefined)
endif

.PHONY: dockerimage
dockerimage:
	docker build --build-arg AWSBEATS_VERSION=$(AWSBEATS_VERSION) --build-arg GO_VERSION=$(GO_VERSION) --build-arg BEATS_VERSION=$(BEATS_VERSION) --build-arg BEAT_NAME=$(BEAT_NAME) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
